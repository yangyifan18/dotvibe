# dotvibe — Vibe Coding Agent Migration & Backup Tool

## Problem

Migrating to a new Mac loses critical AI coding agent data: project memory, custom agents/skills, conversation history, and tool configurations. macOS's built-in migration transfers apps and settings but not the nuanced per-project memory and agent customizations that vibe coding workflows depend on.

Existing solutions (chezmoi, stow, Time Machine) don't understand the internal structure of these tools and can't do selective backup/restore.

## Solution

**dotvibe** — a Go CLI tool that backs up and restores vibe coding agent data across machines. Single binary, no dependencies.

## Supported Tools (v1)

| Tool | Data Dir | Approx Size |
|------|----------|-------------|
| Claude Code | `~/.claude/` | ~533 MB |
| Codex CLI | `~/.codex/` | ~908 MB |
| OpenCode | `~/.config/opencode/` + `~/.opencode/` | ~127 MB |

## Data Model

Each tool has three data categories:

| Category | Description | Default |
|----------|-------------|---------|
| **config** | settings.json, config.toml, tool configs | Included |
| **memory** | MEMORY.md, AGENTS.md, CLAUDE.md, per-project memory | Included |
| **skills** | Custom skills, plugins, agent definitions | Included |
| **history** | Sessions, transcripts, telemetry | Excluded (opt-in) |

### Default Exclusions

- `auth.json`, API keys, credentials
- `telemetry/` directories
- `cache/` directories
- `session-env/` (temporary session data)
- `shell-snapshots/` (temporary)

## CLI Interface

### `dotvibe export`

Create a backup archive.

```
dotvibe export                           # default: config + memory + skills
dotvibe export -o ~/Desktop/my.tar.gz    # custom output path
dotvibe export --with-history            # include session/transcript history
dotvibe export --only claude-code,codex  # backup specific tools only
dotvibe export --exclude "*/Research/*"  # exclude matching paths
dotvibe export --exclude-pattern "transcripts/*.jsonl"
```

Default output: `dotvibe-YYYY-MM-DD.tar.gz` in current directory.

### `dotvibe import <archive>`

Restore from a backup with interactive selection.

```
dotvibe import backup.tar.gz              # interactive selection
dotvibe import backup.tar.gz --yes        # restore everything (non-interactive)
dotvibe import backup.tar.gz --only codex # restore specific tool
dotvibe import backup.tar.gz --project "~/Code/OmniAgent"  # restore specific project
```

Interactive mode shows a tree:
```
Select what to restore:
  [x] Claude Code
    [x] Config (settings.json)
    [x] All projects
      [x] ~/Code/OmniAgent
      [ ] ~/Research
    [x] Skills (3)
  [x] Codex CLI
    [x] Config (config.toml)
    [x] Agents (24)
  [ ] OpenCode
```

### `dotvibe list <archive>`

Show backup contents without extracting.

```
dotvibe list backup.tar.gz
# Output:
# dotvibe backup — 2026-05-19
# Tools: claude-code, codex, opencode
#   Claude Code: config, 12 projects, 3 skills (42 MB)
#   Codex CLI: config, 24 agents (1.2 MB)
#   OpenCode: config (8 KB)
```

### `dotvibe status`

Show detected vibe coding tools on the current machine.

```
$ dotvibe status
Detected vibe coding tools:

  Claude Code    ~/.claude/          533 MB
    ├── 12 projects with memory
    ├── 3 custom skills
    └── 728 transcripts

  Codex CLI      ~/.codex/           908 MB
    ├── 24 custom agents
    ├── config.toml (model: gpt-5.5)
    └── sessions since 2026-01

  OpenCode       ~/.config/opencode/ 127 MB
    └── oh-my-openagent.json
```

## Archive Structure

```
dotvibe-2026-05-19.tar.gz
├── manifest.json              # metadata: version, timestamp, tool versions
├── claude-code/
│   ├── config/
│   │   └── settings.json
│   ├── memory/
│   │   ├── -Users-young-Code-OmniAgent/
│   │   │   ├── MEMORY.md
│   │   │   └── memory/*.md
│   │   └── ...
│   ├── skills/
│   │   └── custom-skill/
│   └── projects/              # per-project session JSONLs (if --with-history)
├── codex-cli/
│   ├── config/
│   │   ├── config.toml
│   │   └── AGENTS.md
│   ├── agents/
│   │   ├── architect.md
│   │   └── ...
│   ├── skills/
│   └── sessions/              # if --with-history
└── opencode/
    └── config/
        └── oh-my-openagent.json
```

### manifest.json

```json
{
  "version": "1.0.0",
  "created": "2026-05-19T12:00:00Z",
  "hostname": "old-macbook",
  "tools": {
    "claude-code": {
      "version": "1.0.x",
      "included": ["config", "memory", "skills"],
      "project_count": 12,
      "file_count": 245
    },
    "codex-cli": {
      "version": "0.x",
      "included": ["config", "agents"],
      "agent_count": 24
    },
    "opencode": {
      "included": ["config"]
    }
  }
}
```

## Technical Design

### Language & Dependencies

- **Go 1.22+** — single binary, no runtime dependencies
- `archive/tar` + `compress/gzip` — standard library
- `github.com/spf13/cobra` — CLI framework
- `github.com/manifoldco/promptui` — interactive selection

### Project Structure

```
dotvibe/
├── cmd/
│   ├── root.go          # cobra root command
│   ├── export.go        # dotvibe export
│   ├── import.go        # dotvibe import
│   ├── list.go          # dotvibe list
│   └── status.go        # dotvibe status
├── adapters/
│   ├── adapter.go       # Adapter interface
│   ├── claude.go        # Claude Code adapter
│   ├── codex.go         # Codex CLI adapter
│   └── opencode.go      # OpenCode adapter
├── backup/
│   ├── archive.go       # tar.gz read/write
│   └── manifest.go      # manifest.json handling
├── config/
│   └── exclude.go       # exclusion rules engine
├── main.go
└── go.mod
```

### Core Interface

```go
type Adapter interface {
    Name() string
    Detect() bool
    ListFiles(opts ExportOpts) []FileEntry
    Restore(entries []FileEntry, dest string) error
    Status() ToolStatus
}

type FileEntry struct {
    SourcePath string
    InArchive  string
    Category   string // "config", "memory", "skills", "history"
    Size       int64
}

type ExportOpts struct {
    WithHistory     bool
    ExcludePatterns []string
    OnlyTools       []string
}

type ToolStatus struct {
    Name       string
    Path       string
    Size       int64
    Projects   int
    Skills     int
    Agents     int
    Sessions   int
    ConfigFile string
}
```

### Adapter Detection

Each adapter detects its tool by checking for the data directory and key files:

- **Claude Code**: `~/.claude/settings.json` exists
- **Codex CLI**: `~/.codex/config.toml` exists
- **OpenCode**: `~/.config/opencode/opencode.json` or `~/.opencode/oh-my-openagent.json` exists

### Restore Behavior

- **Never overwrite existing files** by default — skip with warning
- `--force` flag to overwrite
- Preserve original file permissions
- Adapt paths if username differs (e.g., `/Users/old-user/` → `/Users/new-user/`)
- Show diff summary before applying

## Distribution

- `brew install dotvibe` (tap: `young/tap`)
- `go install github.com/young/dotvibe@latest`
- GitHub Releases (darwin/arm64, darwin/amd64, linux/amd64)

## Future Considerations (out of scope for v1)

- Additional tools: Cursor, Windsurf, Aider, Continue
- Plugin/adapter system for community contributions
- Git repo backup target (in addition to tar.gz)
- Sync mode between multiple Macs
- Encrypted backups (for auth data)
