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
