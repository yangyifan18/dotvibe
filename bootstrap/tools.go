package bootstrap

import "os/exec"

// InstallCommand describes a command a user can run to install a missing tool.
type InstallCommand struct {
	Manager string
	Command string
	SafeRun bool
}

// ToolSpec describes a bootstrap tool and the binaries that indicate it exists.
type ToolSpec struct {
	ID              string
	Name            string
	Binaries        []string
	InstallCommands []InstallCommand
}

// ToolCheckResult reports whether a tool spec was found on PATH.
type ToolCheckResult struct {
	ID              string
	Name            string
	Installed       bool
	FoundBinary     string
	InstallCommands []InstallCommand
}

// DefaultToolSpecs returns the supported coding-agent tools for setup checks.
func DefaultToolSpecs() []ToolSpec {
	return []ToolSpec{
		{
			ID:       "claude-code",
			Name:     "Claude Code",
			Binaries: []string{"claude"},
			InstallCommands: []InstallCommand{
				{Manager: "npm", Command: "npm install -g @anthropic-ai/claude-code", SafeRun: true},
			},
		},
		{
			ID:       "codex-cli",
			Name:     "Codex CLI",
			Binaries: []string{"codex"},
			InstallCommands: []InstallCommand{
				{Manager: "npm", Command: "npm i -g @openai/codex", SafeRun: true},
			},
		},
		{
			ID:       "opencode",
			Name:     "OpenCode",
			Binaries: []string{"opencode"},
			InstallCommands: []InstallCommand{
				{Manager: "brew", Command: "brew install anomalyco/tap/opencode", SafeRun: true},
				{Manager: "npm", Command: "npm i -g opencode-ai", SafeRun: true},
				{Manager: "curl", Command: "curl -fsSL https://opencode.ai/install | bash", SafeRun: false},
			},
		},
	}
}

// DetectTools checks each spec's binaries on PATH while preserving spec order.
func DetectTools(specs []ToolSpec) []ToolCheckResult {
	results := make([]ToolCheckResult, 0, len(specs))
	for _, spec := range specs {
		result := ToolCheckResult{
			ID:              spec.ID,
			Name:            spec.Name,
			InstallCommands: spec.InstallCommands,
		}
		for _, binary := range spec.Binaries {
			path, err := exec.LookPath(binary)
			if err == nil {
				result.Installed = true
				result.FoundBinary = path
				break
			}
		}
		results = append(results, result)
	}
	return results
}
