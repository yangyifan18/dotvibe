<p align="center">
  <img src="https://img.shields.io/badge/dotvibe-v1.0-blue?style=for-the-badge&logo=go" alt="version"/>
  <img src="https://img.shields.io/badge/platform-macOS-lightgrey?style=for-the-badge&logo=apple" alt="platform"/>
  <img src="https://img.shields.io/github/license/yangyifan18/dotvibe?style=for-the-badge" alt="license"/>
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen?style=for-the-badge" alt="prs"/>
</p>

<h1 align="center">dotvibe</h1>

<p align="center">
  <img src="assets/dotvibe.png" alt="dotvibe screenshot" width="760"/>
</p>

<p align="center">
  <b>把你的 vibe coding 环境搬到新 Mac。</b>
</p>

<p align="center">
  <a href="README.md">English</a> &middot;
  <b>中文</b> &middot;
  <a href="https://github.com/yangyifan18/dotvibe/blob/main/LICENSE">License</a>
</p>

---

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
| `diff` | 按路径和 checksum 比较两个备份包 |
| `import` | 选择性恢复 — 按工具、按项目、或全部 |

**支持工具：** Claude Code &middot; Codex CLI &middot; OpenCode

## 安装

```bash
# Go install
go install github.com/yangyifan18/dotvibe@latest

# Homebrew
# 即将支持：brew install yangyifan18/tap/dotvibe

# 从源码构建
git clone https://github.com/yangyifan18/dotvibe.git
cd dotvibe && go build -o dotvibe .
```

## 快速开始

### 给人用

```bash
# 1. 看看本机有什么
dotvibe status

# 2. 备份（默认排除 auth 和 symlink）
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
dotvibe import backup.tar.gz --dry-run --project "~/Code/MyProject"
dotvibe import backup.tar.gz --yes --project "~/Code/MyProject"

# 导出时排除特定路径
dotvibe export --exclude "*/Research/*" --exclude "transcripts/*.jsonl"

# 明确覆盖已有备份
dotvibe export -o backup.tar.gz --force

# 比较两个备份
dotvibe diff old.tar.gz new.tar.gz

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
| **符号链接** | 否 | 跳过，避免跨机器恢复不安全链接 |

## 恢复安全

`import` 会写入目标机器上的 agent 配置目录。建议先 preview；preview 会列出每个文件、目标路径，以及 write / skip / overwrite 动作：

```bash
dotvibe import backup.tar.gz --dry-run
dotvibe import backup.tar.gz --dry-run --project "~/Code/MyProject"
```

做端到端验证时，建议使用临时 HOME，避免写入真实配置：

```bash
HOME=/tmp/dotvibe-restore ./dotvibe import backup.tar.gz --yes
```

项目过滤当前只适用于 Claude Code project memory。Codex CLI 和 OpenCode 按工具级别恢复。新备份包含每文件 SHA-256 checksum；`list`、`diff`、`import` 会在 checksum 存在时拒绝损坏归档。

## 文档

- [Obsidian 文档](obsidian-docs/) — 项目记忆、风险、决策日志
- Release 构建 — `VERSION=0.1.0 ./scripts/build-release.sh`

---

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=yangyifan18/dotvibe&type=Date)](https://star-history.com/#yangyifan18/dotvibe&Date)

## License

[MIT](LICENSE)
