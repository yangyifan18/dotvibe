<p align="center">
  <img src="https://img.shields.io/badge/dotvibe-v1.0-blue?style=for-the-badge&logo=go" alt="version"/>
  <img src="https://img.shields.io/badge/platform-macOS-lightgrey?style=for-the-badge&logo=apple" alt="platform"/>
  <img src="https://img.shields.io/github/license/yangyifan18/dotvibe?style=for-the-badge" alt="license"/>
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen?style=for-the-badge" alt="prs"/>
</p>

<h1 align="center">dotvibe</h1>

<p align="center">
  <b>Migrate your vibe coding setup to a new Mac.</b><br/>
  <b>把你的 vibe coding 环境搬到新 Mac。</b>
</p>

<p align="center">
  <a href="#english">English</a> &middot;
  <a href="#中文">中文</a> &middot;
  <a href="https://github.com/yangyifan18/dotvibe/blob/main/LICENSE">License</a>
</p>

---

<a name="english"></a>

## Why dotvibe?

You spent weeks tuning your AI coding agents — custom memory, project-specific rules, hand-crafted skills, conversation history. Then you get a new Mac and realize:

- **Claude Code**'s project memory (`~/.claude/projects/`) is gone
- **Codex CLI**'s custom agents (`~/.codex/agents/`) are gone
- **OpenCode**'s configs (`~/.config/opencode/`) are gone
- All those carefully written `MEMORY.md`, `AGENTS.md`, `CLAUDE.md` files — vanished

macOS Migration Assistant transfers apps and settings, but not the nuanced per-project data your vibe coding workflow depends on. General dotfile managers (chezmoi, stow) don't understand these tools' internal structure.

**dotvibe** does.

| Feature | Description |
|---|---|
| `status` | Detect all vibe coding tools on your machine |
| `export` | Backup config + memory + skills into a single `.tar.gz` |
| `list` | Inspect what's inside a backup |
| `import` | Restore selectively — by tool, by project, or everything |

**Supported tools:** Claude Code &middot; Codex CLI &middot; OpenCode

## Install

```bash
# Homecoming
brew install yangyifan18/tap/dotvibe

# Go install
go install github.com/young/dotvibe@latest

# Build from source
git clone https://github.com/yangyifan18/dotvibe.git
cd dotvibe && go build -o dotvibe .
```

## Quick Start

### For Humans

```bash
# 1. See what you've got
dotvibe status

# 2. Back it up (auth excluded by default)
dotvibe export

# 3. Send the .tar.gz to your new Mac (AirDrop, scp, whatever)

# 4. On the new machine
dotvibe import dotvibe-2026-05-20.tar.gz
```

### For Agents

```bash
# Agent-friendly: non-interactive, filtered restore
dotvibe import backup.tar.gz --yes --only claude-code

# Restore a specific project's memory
dotvibe import backup.tar.gz --yes --project "~/Code/MyProject"

# Exclude patterns during export
dotvibe export --exclude "*/Research/*" --exclude-pattern "transcripts/*.jsonl"

# Include session history (excluded by default)
dotvibe export --with-history
```

## What Gets Backed Up

| Category | Included | Examples |
|---|---|---|
| **Config** | Yes | `settings.json`, `config.toml`, `AGENTS.md` |
| **Memory** | Yes | `MEMORY.md`, project rules, per-project memory |
| **Skills** | Yes | Custom skills, plugins, agent definitions |
| **Auth** | No | `auth.json`, API keys, tokens |
| **History** | Opt-in | Sessions, transcripts (`--with-history`) |
| **Cache** | No | Telemetry, cache, temp files |

## Documentation

- [Spec](docs/superpowers/specs/) — Design decisions and architecture
- [Implementation Plan](docs/superpowers/plans/) — How it was built (14 tasks, TDD)
- [Obsidian Docs](obsidian-docs/) — Project memory, risks, decisions

---

<a name="中文"></a>

## 为什么需要 dotvibe？

你花了好几周调教你的 AI coding agent — 自定义 memory、项目规则、精心打磨的 skills、对话历史。然后你换了台新 Mac，发现：

- **Claude Code** 的项目记忆（`~/.claude/projects/`）没了
- **Codex CLI** 的自定义 agents（`~/.codex/agents/`）没了
- **OpenCode** 的配置（`~/.config/opencode/`）没了
- 那些精心写的 `MEMORY.md`、`AGENTS.md`、`CLAUDE.md` — 全没了

macOS 迁移助手能转移应用和设置，但转移不了你的 vibe coding 工作流依赖的那些精细数据。通用 dotfile 管理器（chezmoi、stow）不理解这些工具的内部结构。

**dotvibe** 理解。

| 功能 | 说明 |
|---|---|
| `status` | 检测本机所有 vibe coding 工具 |
| `export` | 把 config + memory + skills 打包成一个 `.tar.gz` |
| `list` | 查看备份包里有什么 |
| `import` | 选择性恢复 — 按工具、按项目、或全部 |

**支持工具：** Claude Code &middot; Codex CLI &middot; OpenCode

## 安装

```bash
# Homebrew
brew install yangyifan18/tap/dotvibe

# Go install
go install github.com/young/dotvibe@latest

# 从源码构建
git clone https://github.com/yangyifan18/dotvibe.git
cd dotvibe && go build -o dotvibe .
```

## 快速开始

### 给人用

```bash
# 1. 看看本机有什么
dotvibe status

# 2. 备份（默认排除 auth）
dotvibe export

# 3. 把 .tar.gz 发到新 Mac（AirDrop、scp、随便）

# 4. 在新机器上
dotvibe import dotvibe-2026-05-20.tar.gz
```

### 给 Agent 用

```bash
# 非交互式，只恢复特定工具
dotvibe import backup.tar.gz --yes --only claude-code

# 只恢复某个项目的 memory
dotvibe import backup.tar.gz --yes --project "~/Code/MyProject"

# 导出时排除特定路径
dotvibe export --exclude "*/Research/*" --exclude-pattern "transcripts/*.jsonl"

# 包含会话历史（默认不含）
dotvibe export --with-history
```

## 备份范围

| 类别 | 默认备份 | 示例 |
|---|---|---|
| **配置** | 是 | `settings.json`、`config.toml`、`AGENTS.md` |
| **记忆** | 是 | `MEMORY.md`、项目规则、per-project memory |
| **Skills** | 是 | 自定义 skills、plugins、agent 定义 |
| **Auth** | 否 | `auth.json`、API keys、tokens |
| **历史** | 可选 | 会话、transcripts（`--with-history`） |
| **缓存** | 否 | telemetry、cache、临时文件 |

## 文档

- [设计文档](docs/superpowers/specs/) — 架构决策
- [实现计划](docs/superpowers/plans/) — 14 个 task，TDD 驱动
- [Obsidian 文档](obsidian-docs/) — 项目记忆、风险、决策日志

---

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=yangyifan18/dotvibe&type=Date)](https://star-history.com/#yangyifan18/dotvibe&Date)

## License

[MIT](LICENSE)
