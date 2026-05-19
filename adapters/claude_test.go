package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeAdapter_Detect(t *testing.T) {
	a := &ClaudeAdapter{}

	home, _ := os.UserHomeDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	_, err := os.Stat(settingsPath)
	expected := err == nil

	if got := a.Detect(); got != expected {
		t.Errorf("Detect() = %v, want %v (settings.json exists: %v)", got, expected, err == nil)
	}
}

func TestClaudeAdapter_ListFiles(t *testing.T) {
	a := &ClaudeAdapter{}
	if !a.Detect() {
		t.Skip("Claude Code not installed")
	}

	files := a.ListFiles(ExportOpts{})
	if len(files) == 0 {
		t.Error("ListFiles returned empty — expected at least settings.json")
	}

	found := false
	for _, f := range files {
		if filepath.Base(f.InArchive) == "settings.json" {
			found = true
			if f.Category != CategoryConfig {
				t.Errorf("settings.json category = %q, want %q", f.Category, CategoryConfig)
			}
		}
	}
	if !found {
		t.Error("settings.json not found in file list")
	}
}

func TestClaudeAdapter_Status(t *testing.T) {
	a := &ClaudeAdapter{}
	if !a.Detect() {
		t.Skip("Claude Code not installed")
	}

	s := a.Status()
	if s.Name != "Claude Code" {
		t.Errorf("Name = %q, want %q", s.Name, "Claude Code")
	}
	if s.Path == "" {
		t.Error("Path is empty")
	}
}
