package backup

import (
	"archive/tar"
	"compress/gzip"
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

		if hdr.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return err
			}
			ar.Manifest = &m
		} else if _, err := io.Copy(io.Discard, tr); err != nil {
			return err
		}

		ar.files = append(ar.files, archiveFile{Name: hdr.Name, Size: hdr.Size})
	}
	if ar.Manifest == nil {
		return fmt.Errorf("archive missing manifest.json")
	}
	return nil
}

// ListFiles returns the paths of all files in the archive (excluding manifest).
func (ar *ArchiveReader) ListFiles() []string {
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

// Close releases all resources.
func (ar *ArchiveReader) Close() error {
	return ar.f.Close()
}

// ExtractArchive extracts all files from a tar.gz to a directory.
func ExtractArchive(archivePath, destDir string) error {
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
