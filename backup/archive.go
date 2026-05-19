package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/young/dotvibe/adapters"
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
	gz       *gzip.Reader
	Manifest *Manifest
	files    []archiveFile
}

type archiveFile struct {
	Name string
	Size int64
	Data []byte
}

// ReadArchive opens and indexes an archive.
func ReadArchive(path string) (*ArchiveReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	ar := &ArchiveReader{f: f, gz: gz}
	if err := ar.index(); err != nil {
		f.Close()
		return nil, err
	}

	return ar, nil
}

func (ar *ArchiveReader) index() error {
	tr := tar.NewReader(ar.gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return err
		}

		if hdr.Name == "manifest.json" {
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return err
			}
			ar.Manifest = &m
		}

		ar.files = append(ar.files, archiveFile{Name: hdr.Name, Size: hdr.Size, Data: data})
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
	for _, f := range ar.files {
		if f.Name == name {
			return f.Data, nil
		}
	}
	return nil, fmt.Errorf("file not found in archive: %s", name)
}

// Close releases all resources.
func (ar *ArchiveReader) Close() error {
	ar.gz.Close()
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

		target := filepath.Join(destDir, hdr.Name)

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
