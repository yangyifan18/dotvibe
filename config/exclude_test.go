package config

import (
	"testing"
)

func TestIsExcluded_AuthFiles(t *testing.T) {
	ex := NewExcluder(nil)

	tests := []struct {
		path string
		want bool
	}{
		{"claude-code/config/settings.json", false},
		{"claude-code/config/auth.json", true},
		{"codex-cli/config/config.toml", false},
		{"codex-cli/config/auth.json", true},
		{"codex-cli/config/auth.json.bak-fastrelay-20260505", true},
	}
	for _, tt := range tests {
		got := ex.IsExcluded(tt.path)
		if got != tt.want {
			t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsExcluded_Telemetry(t *testing.T) {
	ex := NewExcluder(nil)

	tests := []struct {
		path string
		want bool
	}{
		{"claude-code/telemetry/event.json", true},
		{"claude-code/telemetry/subdir/data.json", true},
		{"claude-code/memory/file.md", false},
	}
	for _, tt := range tests {
		got := ex.IsExcluded(tt.path)
		if got != tt.want {
			t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsExcluded_Cache(t *testing.T) {
	ex := NewExcluder(nil)

	tests := []struct {
		path string
		want bool
	}{
		{"claude-code/cache/data.json", true},
		{"claude-code/cache/subdir/file", true},
		{"claude-code/config/file.json", false},
	}
	for _, tt := range tests {
		got := ex.IsExcluded(tt.path)
		if got != tt.want {
			t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsExcluded_SessionEnv(t *testing.T) {
	ex := NewExcluder(nil)

	tests := []struct {
		path string
		want bool
	}{
		{"claude-code/session-env/file", true},
		{"claude-code/shell-snapshots/file", true},
		{"claude-code/sessions/file.json", false},
	}
	for _, tt := range tests {
		got := ex.IsExcluded(tt.path)
		if got != tt.want {
			t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsExcluded_CustomPatterns(t *testing.T) {
	ex := NewExcluder([]string{"*/Research/*", "transcripts/*.jsonl"})

	tests := []struct {
		path string
		want bool
	}{
		{"claude-code/memory/-Users-young-Research/MEMORY.md", true},
		{"claude-code/transcripts/abc.jsonl", true},
		{"claude-code/memory/-Users-young-Code/file.md", false},
	}
	for _, tt := range tests {
		got := ex.IsExcluded(tt.path)
		if got != tt.want {
			t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
