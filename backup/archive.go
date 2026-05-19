package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"

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
