# EcoPaste

macOS 优先的本地剪贴板管理器。

项目正在从 **Tauri v2 + Rust + React** 迁移到 **Go + NSGo + AppKit**。当前仓库处于迁移准备阶段，新的 Go/NSGo 应用尚未落地。

## 当前仓库

| 路径 | 内容 |
| --- | --- |
| `docs/nsgo-migration/` | 迁移分析、目标架构、阶段计划、功能对照、风险与清理记录 |
| `legacy/backend/` | 旧剪贴板核心参考代码，不参与构建 |
| `legacy/frontend/` | 旧 React UI 参考，不参与构建 |
| `AGENTS.md` | 项目长期开发规范 |

旧版完整工程与 Tauri 外壳已移出仓库；剪贴板核心和前端 UI 以 `legacy/` 参考保留，归档位置见 `docs/nsgo-migration/06-cleanup-report.md`。

## 目标

- 当前阶段只支持 macOS。
- UI 使用 AppKit 原生控件与系统能力。
- NSGo 负责应用结构、状态、布局和事件编排。
- 保留剪贴板、历史、搜索、分组、收藏、置顶、备注、图片/文件、备份等核心能力。
- 数据契约与旧版兼容，迁移过程可回滚。

## 下一步

按 `docs/nsgo-migration/03-migration-plan.md` 从 Phase 0（兼容夹具）和 Phase 1（Go + NSGo 应用壳）开始。

## License

[Apache License 2.0](./LICENSE)
