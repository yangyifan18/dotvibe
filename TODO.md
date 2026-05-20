# TODO

## Robustness

- [ ] **Restore error handling** — import 遇到 adapter restore 失败时静默继续，可能留下半成品。应 fail-fast 或最后汇总报错并设 non-zero exit code
- [ ] **Export 覆盖警告** — 默认文件名按日期生成，同一天多次 export 会静默覆盖。应检测目标文件已存在时警告或拒绝（除非显式 `--force`）
- [ ] **Import 完整性校验** — manifest 缺少 checksum，tar.gz 传输损坏时 import 会在中途随机失败。应在 manifest 加 sha256，import 时先校验

## Architecture

- [ ] **`--project` filter 泛化** — `cmd/import_cmd.go` 硬编码 `if toolID == "claude-code"`，filter 逻辑应下沉到每个 adapter 自己实现
- [ ] **路径映射显式化** — `adaptPath` 与 `ListFiles` 的 archive 路径格式是隐式耦合，`ListFiles` 改了但 `adaptPath` 没跟上会静默写错位置。应让 `FileEntry` 带 `RestorePath`，或 adapter 统一 path mapping 规则
- [ ] **Export/Import 路径契约** — import 时 `FileEntry.SourcePath` 为空，adapter 靠 `adaptPath` 猜目标路径。export 和 import 之间没有显式契约

## Performance & UX

- [ ] **避免重复 Status() 调用** — export 里 `ListFiles` 已扫描目录，之后又调 `Status()` 拿 project/agent count，应合并或缓存
- [ ] **大备份进度反馈** — 几百个文件时 export 无进度提示，应加 progress bar 或按文件数打点
- [ ] **改进 dry-run 输出** — 目前只打印 tool 级汇总，应列出具体文件（写入/跳过）
- [ ] **Import 完成总结** — restore 结束后无汇总，应报告写入/跳过/失败数量

## Missing Features

- [ ] **Diff/preview** — 无法比较两次 backup 的差异。manifest 记录 file hash 后可实现 `dotvibe diff a.tar.gz b.tar.gz`
- [ ] **Symlink 处理** — `walkDir` 遇到 symlink 当普通文件读，restore 时丢失 symlink 语义。应跳过或保留 symlink 元数据
