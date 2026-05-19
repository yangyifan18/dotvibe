package adapters

import "os"

type ClaudeAdapter struct{}

func NewClaudeAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{}
}

func (a *ClaudeAdapter) Name() string { return "Claude Code" }
func (a *ClaudeAdapter) ID() string   { return "claude-code" }

func (a *ClaudeAdapter) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(home + "/.claude/settings.json")
	return err == nil
}

func (a *ClaudeAdapter) ListFiles(opts ExportOpts) []FileEntry { return nil }
func (a *ClaudeAdapter) Status() ToolStatus                    { return ToolStatus{} }
func (a *ClaudeAdapter) RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) error {
	return nil
}
