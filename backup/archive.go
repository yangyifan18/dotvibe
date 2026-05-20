package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yangyifan18/dotvibe/adapters"
)

// CreateArchive writes a tar.gz file containing manifest and all entries.
func CreateArchive(dst string, manifest *Manifest, entries []adapters.FileEntry) error {
	files, err := buildFileManifest(entries)
	if err != nil {
		return err
	}
	manifest.Files = files

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write manifest first
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := writeBytesToTar(tw, "manifest.json", manifestData); err != nil {
		return err
	}

	// Write all file entries
	for _, entry := range entries {
		if err := writeFileToTar(tw, entry.SourcePath, entry.InArchive); err != nil {
			return err
		}
	}

	return nil
}

func writeBytesToTar(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func writeFileToTar(tw *tar.Writer, srcPath, archivePath string) error {
	if err := validateArchivePath(archivePath); err != nil {
		return err
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr := &tar.Header{
		Name:    archivePath,
		Mode:    int64(info.Mode()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	_, err = io.Copy(tw, f)
	return err
}

// ReadArchive opens a tar.gz for reading. Caller must call Close().
type ArchiveReader struct {
	f        *os.File
	path     string
	Manifest *Manifest
	files    []archiveFile
}

type archiveFile struct {
	Name string
	Size int64
}

// ReadArchive opens and indexes an archive.
func ReadArchive(path string) (*ArchiveReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	ar := &ArchiveReader{f: f, path: path}
	if err := ar.index(); err != nil {
		f.Close()
		return nil, err
	}

	return ar, nil
}

func (ar *ArchiveReader) index() error {
	if _, err := ar.f.Seek(0, 0); err != nil {
		return err
	}
	gz, err := gzip.NewReader(ar.f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	seen := map[string]struct{}{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := validateTarHeader(hdr); err != nil {
			return err
		}
		if _, ok := seen[hdr.Name]; ok {
			return fmt.Errorf("duplicate archive entry: %s", hdr.Name)
		}
		seen[hdr.Name] = struct{}{}

		if hdr.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return err
			}
			m.Normalize()
			ar.Manifest = &m
		} else if _, err := io.Copy(io.Discard, tr); err != nil {
			return err
		}

		ar.files = append(ar.files, archiveFile{Name: hdr.Name, Size: hdr.Size})
	}
	if ar.Manifest == nil {
		return fmt.Errorf("archive missing manifest.json")
	}
	if len(ar.Manifest.Files) > 0 {
		if err := ar.verifyManifestFiles(); err != nil {
			return err
		}
	}
	return nil
}

// ListFiles returns the paths of all files in the archive (excluding manifest).
func (ar *ArchiveReader) ListFiles() []string {
	if ar.Manifest != nil && len(ar.Manifest.Files) > 0 {
		names := make([]string, 0, len(ar.Manifest.Files))
		for _, file := range ar.Manifest.Files {
			names = append(names, file.Path)
		}
		return names
	}

	var names []string
	for _, f := range ar.files {
		if f.Name != "manifest.json" {
			names = append(names, f.Name)
		}
	}
	return names
}

// ReadFile returns the contents of a file from the archive.
func (ar *ArchiveReader) ReadFile(name string) ([]byte, error) {
	if err := validateArchivePath(name); err != nil {
		return nil, err
	}
	physicalName, err := ar.physicalPathForRead(name)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(ar.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := validateTarHeader(hdr); err != nil {
			return nil, err
		}
		if hdr.Name == physicalName {
			return io.ReadAll(tr)
		}
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("file not found in archive: %s", name)
}

func (ar *ArchiveReader) physicalPathForRead(name string) (string, error) {
	if ar.Manifest == nil {
		return name, nil
	}
	for _, file := range ar.Manifest.Files {
		if file.Path != name {
			continue
		}
		if file.Storage == FileStorageBase {
			return "", fmt.Errorf("file stored in base archive: %s", name)
		}
		physicalPath := storedPathForInlineFile(file)
		if err := validateArchivePath(physicalPath); err != nil {
			return "", err
		}
		return physicalPath, nil
	}
	return name, nil
}

// Close releases all resources.
func (ar *ArchiveReader) Close() error {
	return ar.f.Close()
}

func (ar *ArchiveReader) verifyManifestFiles() error {
	expected := map[string]FileManifest{}
	allowedPayloads := map[string]struct{}{}
	for _, file := range ar.Manifest.Files {
		if err := validateArchivePath(file.Path); err != nil {
			return err
		}
		expected[file.Path] = file
		if file.Storage != FileStorageBase {
			physicalPath := storedPathForInlineFile(file)
			if err := validateArchivePath(physicalPath); err != nil {
				return err
			}
			allowedPayloads[physicalPath] = struct{}{}
		}
	}

	actual := map[string]archiveFile{}
	for _, file := range ar.files {
		if file.Name != "manifest.json" {
			if _, ok := allowedPayloads[file.Name]; !ok {
				return fmt.Errorf("archive contains payload not listed in manifest: %s", file.Name)
			}
			actual[file.Name] = file
		}
	}

	for path, want := range expected {
		if want.Storage == FileStorageBase {
			continue
		}
		physicalPath := storedPathForInlineFile(want)
		if err := validateArchivePath(physicalPath); err != nil {
			return err
		}
		got, ok := actual[physicalPath]
		if !ok {
			return fmt.Errorf("manifest file missing from archive: %s", path)
		}
		if got.Size != want.Size {
			return fmt.Errorf("archive file size mismatch for %s: got %d, want %d", path, got.Size, want.Size)
		}
		sum, err := ar.fileSHA256(path)
		if err != nil {
			return err
		}
		if sum != want.SHA256 {
			return fmt.Errorf("archive file checksum mismatch for %s", path)
		}
	}
	return nil
}

func storedPathForInlineFile(file FileManifest) string {
	if file.StoredPath != "" {
		return file.StoredPath
	}
	return file.Path
}

func (ar *ArchiveReader) fileSHA256(name string) (string, error) {
	data, err := ar.ReadFile(name)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

// ExtractArchive extracts all files from a tar.gz to a directory.
func ExtractArchive(archivePath, destDir string) error {
	ar, err := ReadArchive(archivePath)
	if err != nil {
		return err
	}
	if len(ar.Manifest.Files) > 0 {
		defer ar.Close()
		return ar.extractManifestFiles(destDir)
	}
	if err := ar.Close(); err != nil {
		return err
	}
	return extractRawArchive(archivePath, destDir)
}

func (ar *ArchiveReader) extractManifestFiles(destDir string) error {
	for _, file := range ar.Manifest.Files {
		if file.Storage == FileStorageBase {
			continue
		}
		target, err := safeExtractTarget(destDir, file.Path)
		if err != nil {
			return err
		}
		data, err := ar.ReadFile(file.Path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func extractRawArchive(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := validateTarHeader(hdr); err != nil {
			return err
		}

		target, err := safeExtractTarget(destDir, hdr.Name)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		outFile, err := os.Create(target)
		if err != nil {
			return err
		}

		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	return nil
}

func validateTarHeader(hdr *tar.Header) error {
	if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
		return fmt.Errorf("unsupported archive entry type for %q", hdr.Name)
	}
	return validateArchivePath(hdr.Name)
}

func validateArchivePath(name string) error {
	if name == "" {
		return fmt.Errorf("archive entry has empty path")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return fmt.Errorf("archive entry uses absolute path: %s", name)
	}
	if strings.Contains(name, "\\") {
		return fmt.Errorf("archive entry uses backslash path: %s", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("archive entry has unsafe path component: %s", name)
		}
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("archive entry escapes destination: %s", name)
	}
	return nil
}

func safeExtractTarget(destDir, name string) (string, error) {
	if err := validateArchivePath(name); err != nil {
		return "", err
	}
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	target := filepath.Join(destAbs, filepath.Clean(name))
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if targetAbs != destAbs && !strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes destination: %s", name)
	}
	return targetAbs, nil
}

func buildFileManifest(entries []adapters.FileEntry) ([]FileManifest, error) {
	files := make([]FileManifest, 0, len(entries))
	for _, entry := range entries {
		if err := validateArchivePath(entry.InArchive); err != nil {
			return nil, err
		}
		info, err := os.Stat(entry.SourcePath)
		if err != nil {
			return nil, err
		}
		sum, err := sourceFileSHA256(entry.SourcePath)
		if err != nil {
			return nil, err
		}
		files = append(files, FileManifest{
			Path:     entry.InArchive,
			Size:     info.Size(),
			SHA256:   sum,
			Category: entry.Category,
		})
	}
	return files, nil
}

func sourceFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
