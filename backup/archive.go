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
	"sort"
	"strings"

	"github.com/yangyifan18/dotvibe/adapters"
)

// CreateArchive writes a tar.gz file containing manifest and all entries.
func CreateArchive(dst string, manifest *Manifest, entries []adapters.FileEntry) error {
	plan, err := BuildFullArchivePlan(manifest, entries)
	if err != nil {
		return err
	}
	return CreateArchiveWithStoredEntries(dst, plan.Manifest, plan.StoredEntries)
}

// StoredEntry maps a source file to its physical storage path inside an archive.
type StoredEntry struct {
	SourcePath string
	StoredPath string
}

// CreateArchiveWithStoredEntries writes a tar.gz file using manifest logical paths
// and independent physical storage paths for payloads.
func CreateArchiveWithStoredEntries(dst string, manifest *Manifest, entries []StoredEntry) error {
	if err := validateStoredArchiveInputs(manifest, entries); err != nil {
		return err
	}

	return createArchiveFile(dst, manifest, func(tw *tar.Writer) error {
		for _, entry := range entries {
			if err := writeFileToTar(tw, entry.SourcePath, entry.StoredPath); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateStoredArchiveInputs(manifest *Manifest, entries []StoredEntry) error {
	if manifest == nil {
		return fmt.Errorf("manifest is nil")
	}
	manifest.Normalize()

	type storedSource struct {
		size   int64
		sha256 string
	}
	entryByPath := map[string]storedSource{}
	for _, entry := range entries {
		if err := validateArchivePath(entry.StoredPath); err != nil {
			return err
		}
		if _, ok := entryByPath[entry.StoredPath]; ok {
			return fmt.Errorf("duplicate stored archive path: %s", entry.StoredPath)
		}
		info, err := os.Stat(entry.SourcePath)
		if err != nil {
			return err
		}
		sum, err := sourceFileSHA256(entry.SourcePath)
		if err != nil {
			return err
		}
		entryByPath[entry.StoredPath] = storedSource{size: info.Size(), sha256: sum}
	}

	logicalPaths := map[string]struct{}{}
	expectedStoredPaths := map[string]struct{}{}
	for _, file := range manifest.Files {
		if err := validateArchivePath(file.Path); err != nil {
			return err
		}
		if _, ok := logicalPaths[file.Path]; ok {
			return fmt.Errorf("duplicate logical manifest path: %s", file.Path)
		}
		logicalPaths[file.Path] = struct{}{}

		switch file.Storage {
		case FileStorageInline:
			storedPath := storedPathForInlineFile(file)
			if err := validateArchivePath(storedPath); err != nil {
				return err
			}
			source, ok := entryByPath[storedPath]
			if !ok {
				return fmt.Errorf("manifest inline file missing stored payload: %s", file.Path)
			}
			if source.size != file.Size {
				return fmt.Errorf("stored source size mismatch for %s: got %d, want %d", file.Path, source.size, file.Size)
			}
			if source.sha256 != file.SHA256 {
				return fmt.Errorf("stored source checksum mismatch for %s", file.Path)
			}
			expectedStoredPaths[storedPath] = struct{}{}
		case FileStorageBase:
			// Base-backed files describe logical state but do not store payloads.
		default:
			return fmt.Errorf("unsupported file storage for %s: %s", file.Path, file.Storage)
		}
	}

	for storedPath := range entryByPath {
		if _, ok := expectedStoredPaths[storedPath]; !ok {
			return fmt.Errorf("stored payload not listed in manifest: %s", storedPath)
		}
	}
	return nil
}

func createArchiveFile(dst string, manifest *Manifest, writePayloads func(*tar.Writer) error) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write manifest first
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return closeArchiveWriters(tw, gw, f, err)
	}
	if err := writeBytesToTar(tw, "manifest.json", manifestData); err != nil {
		return closeArchiveWriters(tw, gw, f, err)
	}
	if err := writePayloads(tw); err != nil {
		return closeArchiveWriters(tw, gw, f, err)
	}

	return closeArchiveWriters(tw, gw, f, nil)
}

func closeArchiveWriters(tw *tar.Writer, gw *gzip.Writer, f *os.File, firstErr error) error {
	if err := tw.Close(); firstErr == nil && err != nil {
		firstErr = err
	}
	if err := gw.Close(); firstErr == nil && err != nil {
		firstErr = err
	}
	if err := f.Close(); firstErr == nil && err != nil {
		firstErr = err
	}
	return firstErr
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
	f             *os.File
	path          string
	Manifest      *Manifest
	files         []archiveFile
	storedByPath  map[string]archiveFile
	logicalByPath map[string]FileManifest
}

type archiveFile struct {
	Name   string
	Size   int64
	SHA256 string
}

// ReadArchive opens and indexes an archive.
func ReadArchive(path string) (*ArchiveReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	ar := &ArchiveReader{
		f:             f,
		path:          path,
		storedByPath:  map[string]archiveFile{},
		logicalByPath: map[string]FileManifest{},
	}
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
		} else {
			h := sha256.New()
			if _, err := io.Copy(io.MultiWriter(io.Discard, h), tr); err != nil {
				return err
			}
			file := archiveFile{Name: hdr.Name, Size: hdr.Size, SHA256: fmt.Sprintf("%x", h.Sum(nil))}
			ar.files = append(ar.files, file)
			ar.storedByPath[hdr.Name] = file
			continue
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
		names := make([]string, 0, len(ar.logicalByPath))
		for name := range ar.logicalByPath {
			names = append(names, name)
		}
		sort.Strings(names)
		return names
	}

	var names []string
	for _, f := range ar.files {
		if f.Name != "manifest.json" {
			names = append(names, f.Name)
		}
	}
	sort.Strings(names)
	return names
}

// ReadFile returns the contents of a file from the archive.
func (ar *ArchiveReader) ReadFile(name string) ([]byte, error) {
	if err := validateArchivePath(name); err != nil {
		return nil, err
	}
	physicalName, err := ar.storedPathForRead(name)
	if err != nil {
		return nil, err
	}
	return ar.readStoredFile(physicalName)
}

func (ar *ArchiveReader) readStoredFile(name string) ([]byte, error) {
	if err := validateArchivePath(name); err != nil {
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
		if hdr.Name == name {
			return io.ReadAll(tr)
		}
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("file not found in archive: %s", name)
}

func (ar *ArchiveReader) storedPathForRead(name string) (string, error) {
	if ar.Manifest == nil || len(ar.Manifest.Files) == 0 {
		return name, nil
	}
	file, ok := ar.logicalByPath[name]
	if !ok {
		return "", fmt.Errorf("file not listed in archive manifest: %s", name)
	}
	switch file.Storage {
	case FileStorageInline:
	case FileStorageBase:
		return "", fmt.Errorf("file stored in base archive: %s", name)
	default:
		return "", fmt.Errorf("unsupported file storage for %s: %s", name, file.Storage)
	}
	physicalPath := storedPathForInlineFile(file)
	if err := validateArchivePath(physicalPath); err != nil {
		return "", err
	}
	return physicalPath, nil
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
		if _, ok := expected[file.Path]; ok {
			return fmt.Errorf("duplicate logical manifest path: %s", file.Path)
		}
		expected[file.Path] = file
		ar.logicalByPath[file.Path] = file
		switch file.Storage {
		case FileStorageInline:
			physicalPath := storedPathForInlineFile(file)
			if err := validateArchivePath(physicalPath); err != nil {
				return err
			}
			allowedPayloads[physicalPath] = struct{}{}
		case FileStorageBase:
		default:
			return fmt.Errorf("unsupported file storage for %s: %s", file.Path, file.Storage)
		}
	}

	for _, file := range ar.files {
		if file.Name != "manifest.json" {
			if _, ok := allowedPayloads[file.Name]; !ok {
				return fmt.Errorf("archive contains payload not listed in manifest: %s", file.Name)
			}
		}
	}

	for path, want := range expected {
		switch want.Storage {
		case FileStorageInline:
		case FileStorageBase:
			continue
		default:
			return fmt.Errorf("unsupported file storage for %s: %s", path, want.Storage)
		}
		physicalPath := storedPathForInlineFile(want)
		if err := validateArchivePath(physicalPath); err != nil {
			return err
		}
		got, ok := ar.storedByPath[physicalPath]
		if !ok {
			return fmt.Errorf("manifest file missing from archive: %s", path)
		}
		if got.Size != want.Size {
			return fmt.Errorf("archive file size mismatch for %s: got %d, want %d", path, got.Size, want.Size)
		}
		if got.SHA256 != want.SHA256 {
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

// ExtractArchive extracts all files from a tar.gz to a directory.
func ExtractArchive(archivePath, destDir string) error {
	ar, err := ReadArchive(archivePath)
	if err != nil {
		return err
	}
	defer ar.Close()
	return ar.extractManifestFiles(destDir)
}

func (ar *ArchiveReader) extractManifestFiles(destDir string) error {
	for _, name := range ar.ListFiles() {
		target, err := safeExtractTarget(destDir, name)
		if err != nil {
			return err
		}
		data, err := ar.ReadFile(name)
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
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if err := validateArchivePath(entry.InArchive); err != nil {
			return nil, err
		}
		if _, ok := seen[entry.InArchive]; ok {
			return nil, fmt.Errorf("duplicate archive path: %s", entry.InArchive)
		}
		seen[entry.InArchive] = struct{}{}
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
