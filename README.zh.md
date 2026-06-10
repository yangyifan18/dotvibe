<p align="center">
  <img src="https://img.shields.io/badge/dotvibe-v1.1-blue?style=for-the-badge&logo=go" alt="version"/>
  <img src="https://img.shields.io/badge/platform-macOS-lightgrey?style=for-the-badge&logo=apple" alt="platform"/>
  <img src="https://img.shields.io/github/license/yangyifan18/dotvibe?style=for-the-badge" alt="license"/>
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen?style=for-the-badge" alt="prs"/>
</p>

<h1 align="center">dotvibe</h1>

<p align="center">
  <img src="assets/teaserv1.png" alt="dotvibe teaser" width="760"/>
</p>

<p align="center">
  <b>安全备份和迁移 Claude Code、Codex CLI、OpenCode 的记忆。</b>
  <br/>
  在机器之间迁移项目 memory、自定义 agents、skills、规则和 settings；默认不复制 auth token 或私人 transcripts。
</p>

<p align="center">
  <a href="README.md">English</a> &middot;
  <b>中文</b> &middot;
  <a href="https://github.com/yangyifan18/dotvibe/blob/main/LICENSE">License</a>
</p>

---

## 为什么需要 dotvibe？

AI coding 工具已经有了真正的工作记忆：项目规则、Claude Code 项目 memory、Codex agents、自定义 skills、全局 prompts 和安全 settings。通用 dotfile 管理器只能看到路径，却不知道哪些文件是隐私数据、哪些适合分享，也不知道恢复前应该如何审查。

这些场景适合用 dotvibe：

- 把 Claude Code / Codex CLI / OpenCode 上下文搬到新机器或重建后的开发环境；
- 写入真实 agent 目录前，先审查将要恢复什么；
- 比较两次备份，确认 AI coding memory 发生了哪些变化；
- 把 agents、skills、全局规则打包成 `.vibe` recipe 分享，同时避免泄漏私人项目 memory；
- 让 agent 通过 dry-run、stage 和项目 memory merge review 引导导出/导入。

## 和 macOS 迁移助理有什么区别？

macOS 迁移助理适合把整台 Mac 迁到另一台 Mac。dotvibe 更窄、更 agent-native：它只关注 AI coding 上下文，支持导出前审查、按工具/项目选择性导入、处理用户名或项目路径变化、stage 后合并项目 memory，以及生成不包含私人 transcripts / 项目 memory 的 `.vibe` recipe。

如果你已经用整机迁移把所有内容完整搬过去，并且 Claude Code / Codex / OpenCode 都正常工作，那这台机器上可能不需要 dotvibe。dotvibe 更适合新环境初始化、局部迁移、团队 recipe、远程/开发容器环境、备份、diff 和可控恢复。

| 功能 | 说明 |
|---|---|
| `status` | 检测本机所有 vibe coding 工具 |
| `export` | 把 config + memory + skills 打包成一个 `.tar.gz` |
| `list` | 查看备份包里有什么 |
| `diff` | 按 manifest 路径、checksum、工具或分类比较两个备份包 |
| `setup` | 在新机器/环境上检测/安装支持的 agent CLI，然后可选恢复备份 |
| `import` | 选择性恢复 — 按工具、按项目、或全部 |
| `recipe export` | 导出只包含 skills、agents、全局规则和安全 settings 的 `.vibe` 配方 |
| `recipe apply` | 应用 `.vibe` 配方，不需要完整备份 |

**支持工具：** Claude Code &middot; Codex CLI &middot; OpenCode

## 60 秒试用

```bash
brew install yangyifan18/tap/dotvibe
dotvibe status
dotvibe export -o dotvibe-backup.tar.gz
dotvibe list dotvibe-backup.tar.gz
```

默认情况下，dotvibe 会跳过 auth 文件、API keys、telemetry、cache、symlink 和会话历史。写入真实配置前，建议先跑 `dotvibe import --dry-run` 或 `dotvibe import --stage`。

## 安装

```bash
# Go install
go install github.com/yangyifan18/dotvibe@latest

# Homebrew
brew install yangyifan18/tap/dotvibe

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

# 3. 把 .tar.gz 发到新机器/环境（AirDrop、scp、随便）

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

# 创建完整备份
dotvibe export -o dotvibe-full.tar.gz

# 基于旧备份创建增量备份
dotvibe export --base dotvibe-full.tar.gz -o dotvibe-delta.tar.gz

# 恢复增量备份链
dotvibe import dotvibe-delta.tar.gz --base dotvibe-full.tar.gz --yes

# 比较两个备份
dotvibe diff old.tar.gz new.tar.gz

# 查看两个备份之间的 Claude Code memory 变化
dotvibe diff --only claude-code --category memory old.tar.gz new.tar.gz

# 输出 JSON 供自动化使用
dotvibe diff --json old.tar.gz new.tar.gz

# 包含会话历史（默认不含）
dotvibe export --with-history
```

## Agent 辅助迁移

DotVibe 的推荐用法是交给 agent 驱动。你不需要记住所有命令；安装 `agent/codex/dotvibe-migration/` 里的 Codex skill 后，对 agent 说：

```bash
CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
mkdir -p "$CODEX_HOME/skills/dotvibe-migration"
cp -R agent/codex/dotvibe-migration/. "$CODEX_HOME/skills/dotvibe-migration/"
```

重启 Codex 或重新加载 skills，然后说：

```text
调用 dotvibe 导出记忆。
```

Agent 会调用 `dotvibe agent doctor --json`、`dotvibe agent inventory --json`、导出/导入 plan、dry-run 和 staging workspace，引导你做选择。遇到项目 memory 冲突时，agent 会先把归档版本和本地版本 stage 出来，让你 review 或 merge，再写入真实工具目录。

对于项目 memory，agent 辅助导入会处理跨机器后的用户名和 home 路径变化。比如旧备份来自 `/Users/young/Softwares/dotvibe`，新机器会规划到 `/Users/youtopia/Softwares/dotvibe`。如果新机器还没有这个项目，agent 会展示备份 metadata 里的 sanitized `git clone` 命令并先询问；如果项目已存在，agent 会先询问是否加载旧 memory，遇到冲突则 stage 后 review。

## Vibe Recipes

Recipe 用来把一套 vibe coding 配置分享给同事或社区。它会剥离个人数据，只保留 Claude Code skills、Codex agents、全局规则和安全 settings 等可共享内容。

```bash
# 导出 recipe
dotvibe recipe export --name "YYF Vibe Stack" --author "yangyifan" -o yyf.vibe

# 分享或应用前先 inspect / lint
dotvibe recipe inspect yyf.vibe
dotvibe recipe lint yyf.vibe --strict

# 比较两个 recipe
dotvibe recipe diff old.vibe new.vibe
dotvibe recipe diff old.vibe new.vibe --content

# 带 lint gate 和冲突处理地安全应用
dotvibe recipe apply yyf.vibe --dry-run
dotvibe recipe apply yyf.vibe

# 非交互模式
dotvibe recipe apply yyf.vibe --yes              # 跳过冲突
dotvibe recipe apply yyf.vibe --force --yes      # 覆盖冲突
dotvibe recipe apply yyf.vibe --allow-risk       # lint error 仍继续

# 回滚一次 recipe apply
dotvibe rollback list
dotvibe rollback 20260521-143012-a1b2c3
dotvibe rollback 20260521-143012-a1b2c3 --path ~/.codex/agents/reviewer.md
dotvibe rollback prune --keep 20 --dry-run
```

Recipe 不包含 Claude 项目 memory、transcripts、Codex sessions、auth 文件、telemetry 或 cache 数据。`recipe apply` 会先运行 lint，并为 write / overwrite 记录 rollback。顶层 `dotvibe apply` 仅保留为 deprecated 兼容别名。

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

## Setup 安全模型

`dotvibe setup` 默认只预览，不会安装。它会打印已检测工具和安装命令。加 `--install` 后才会在确认后运行安全的包管理器命令；标记为 manual-review 的命令只展示，不自动执行。

```bash
# 预览已检测工具、安装命令和可选恢复计划
dotvibe setup backup.tar.gz

# 跳过确认，运行安全安装命令，然后恢复
dotvibe setup backup.tar.gz --install --yes
```

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
- [Homebrew Preflight](docs/homebrew-preflight.md) — Homebrew formula 变更的维护者验证流程
- [v1.1 Release Notes](docs/releases/v1.1.md)
- Release 构建 — `VERSION=1.1 ./scripts/build-release.sh`

---

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=yangyifan18/dotvibe&type=Date)](https://star-history.com/#yangyifan18/dotvibe&Date)

## License

[MIT](LICENSE)
