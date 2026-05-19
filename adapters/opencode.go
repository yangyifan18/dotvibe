package adapters

import "os"

type OpenCodeAdapter struct{}

func NewOpenCodeAdapter() *OpenCodeAdapter {
	return &OpenCodeAdapter{}
}

func (a *OpenCodeAdapter) Name() string { return "OpenCode" }
func (a *OpenCodeAdapter) ID() string   { return "opencode" }

func (a *OpenCodeAdapter) Detect() bool {
	home, _ := os.UserHomeDir()
	paths := []string{
		home + "/.config/opencode/opencode.json",
		home + "/.opencode/oh-my-openagent.json",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func (a *OpenCodeAdapter) ListFiles(opts ExportOpts) []FileEntry { return nil }
func (a *OpenCodeAdapter) Status() ToolStatus                    { return ToolStatus{} }
func (a *OpenCodeAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) error {
	return nil
}
