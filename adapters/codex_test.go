package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexAdapter_Detect(t *testing.T) {
	a := &CodexAdapter{}
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	_, err := os.Stat(configPath)
	expected := err == nil

	if got := a.Detect(); got != expected {
		t.Errorf("Detect() = %v, want %v", got, expected)
	}
}

func TestCodexAdapter_ListFiles(t *testing.T) {
	a := &CodexAdapter{}
	if !a.Detect() {
		t.Skip("Codex CLI not installed")
	}

	files := a.ListFiles(ExportOpts{})
	if len(files) == 0 {
		t.Error("ListFiles returned empty")
	}

	found := false
	for _, f := range files {
		if filepath.Base(f.InArchive) == "config.toml" {
			found = true
		}
	}
	if !found {
		t.Error("config.toml not found in file list")
	}
}

func TestCodexAdapter_Status(t *testing.T) {
	a := &CodexAdapter{}
	if !a.Detect() {
		t.Skip("Codex CLI not installed")
	}

	s := a.Status()
	if s.Name != "Codex CLI" {
		t.Errorf("Name = %q, want %q", s.Name, "Codex CLI")
	}
}
