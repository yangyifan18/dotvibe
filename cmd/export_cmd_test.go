package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExportRefusesExistingOutputWithoutForce(t *testing.T) {
	oldOutput, oldForce, oldOnly, oldHist, oldExcludes := exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes
	defer func() {
		exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes = oldOutput, oldForce, oldOnly, oldHist, oldExcludes
	}()
	path := filepath.Join(t.TempDir(), "existing.tar.gz")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	exportOutput = path
	exportForce = false
	exportOnly = ""
	exportWithHist = false
	exportExcludes = nil

	if err := exportCmd.RunE(exportCmd, nil); err == nil {
		t.Fatal("expected export to refuse existing output without --force")
	}
}

func TestExportAllowsExistingOutputWithForce(t *testing.T) {
	oldOutput, oldForce, oldOnly, oldHist, oldExcludes := exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes
	defer func() {
		exportOutput, exportForce, exportOnly, exportWithHist, exportExcludes = oldOutput, oldForce, oldOnly, oldHist, oldExcludes
	}()
	path := filepath.Join(t.TempDir(), "existing.tar.gz")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	exportOutput = path
	exportForce = true
	exportOnly = ""
	exportWithHist = false
	exportExcludes = nil

	if err := exportCmd.RunE(exportCmd, nil); err != nil {
		t.Fatalf("expected forced export to overwrite existing output: %v", err)
	}
}

func TestIsExportableFileSkipsSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if !isExportableFile(target) {
		t.Fatal("regular file should be exportable")
	}
	if isExportableFile(link) {
		t.Fatal("symlink should not be exportable")
	}
}
