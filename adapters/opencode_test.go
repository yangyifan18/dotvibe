package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCodeAdapter_Detect(t *testing.T) {
	a := &OpenCodeAdapter{}
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, ".config", "opencode", "opencode.json"),
		filepath.Join(home, ".opencode", "oh-my-openagent.json"),
	}
	expected := false
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			expected = true
		}
	}

	if got := a.Detect(); got != expected {
		t.Errorf("Detect() = %v, want %v", got, expected)
	}
}

func TestOpenCodeAdapter_ListFiles(t *testing.T) {
	a := &OpenCodeAdapter{}
	if !a.Detect() {
		t.Skip("OpenCode not installed")
	}

	files := a.ListFiles(ExportOpts{})
	if len(files) == 0 {
		t.Error("ListFiles returned empty")
	}
}

func TestOpenCodeAdapter_PreservesSourceConfigRoots(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".config", "opencode", "opencode.json"), "config")
	writeTestFile(t, filepath.Join(home, ".opencode", "oh-my-openagent.json"), "legacy")

	a := &OpenCodeAdapter{home: home}
	entries := a.ListFiles(ExportOpts{})
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.InArchive] = true
	}

	if !seen["opencode/xdg-config/opencode.json"] {
		t.Fatalf("missing xdg config archive path; got %#v", seen)
	}
	if !seen["opencode/home-config/oh-my-openagent.json"] {
		t.Fatalf("missing home config archive path; got %#v", seen)
	}
}
