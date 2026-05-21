package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexAdapter_Detect(t *testing.T) {
	tests := []struct {
		name  string
		setup func(string)
		want  bool
	}{
		{
			name: "config file",
			setup: func(home string) {
				writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), `model = "test"`)
			},
			want: true,
		},
		{
			name: "agents directory",
			setup: func(home string) {
				if err := os.MkdirAll(filepath.Join(home, ".codex", "agents"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			want: true,
		},
		{name: "empty home", setup: func(string) {}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			tt.setup(home)
			a := &CodexAdapter{home: home}
			if got := a.Detect(); got != tt.want {
				t.Fatalf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodexAdapter_ListFiles(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), `model = "test"`)
	writeTestFile(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "agent")

	a := &CodexAdapter{home: home}
	files := a.ListFiles(ExportOpts{})
	assertArchiveEntry(t, files, "codex-cli/config/config.toml", CategoryConfig)
	assertArchiveEntry(t, files, "codex-cli/agents/reviewer.md", CategorySkills)
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

func TestCodexAdapter_DetectsAgentsDirectoryWithoutConfig(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".codex", "agents", "reviewer.md"), "agent")

	a := &CodexAdapter{home: home}
	if !a.Detect() {
		t.Fatal("expected agents directory to detect Codex CLI data")
	}
}
