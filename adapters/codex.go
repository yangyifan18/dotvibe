package adapters

import "os"

type CodexAdapter struct{}

func NewCodexAdapter() *CodexAdapter {
	return &CodexAdapter{}
}

func (a *CodexAdapter) Name() string { return "Codex CLI" }
func (a *CodexAdapter) ID() string   { return "codex-cli" }

func (a *CodexAdapter) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(home + "/.codex/config.toml")
	return err == nil
}

func (a *CodexAdapter) ListFiles(opts ExportOpts) []FileEntry { return nil }
func (a *CodexAdapter) Status() ToolStatus                    { return ToolStatus{} }
func (a *CodexAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) error {
	return nil
}
