# EcoPaste × NSGo 迁移文档

本目录记录 EcoPaste 从 **Tauri v2 + React 19 + Rust** 迁移到 **Go + NSGo + AppKit** 的前期分析与渐进式迁移计划。

## 当前结论

- [2.0.0 是新项目的首个发布与兼容性基线](../../AGENTS.md#51-版本兼容策略)。低于 2.0.0 的配置、数据和备份不迁移、不兼容，用户自行清理；旧项目仅用于参考。
- NSGo 是 Go 原生 UI 引擎，不是 Tauri 的替代前端框架。接入 NSGo 意味着应用运行时从 Rust/Tauri 切换为 Go。
- 现有 Rust 侧约 22,580 行、前端约 20,687 行。剪贴板、数据库、设置、备份、窗口和系统集成等核心能力都在 Rust，不能直接复用为 NSGo 组件。
- 推荐路线是 **Go 重写核心 + NSGo 负责应用结构/状态/布局 + AppKit 原生控件负责 macOS UI**，而不是把 Rust 作为 sidecar 或 cgo 静态库长期保留。
- 迁移必须分阶段进行。迁移前的完整实现已归档，仓库保留 `legacy/backend/` 剪贴板核心与 `legacy/frontend/` UI 参考。
- 当前阶段只保证 macOS。Windows/Linux 旧代码不保留在仓库，需要时从归档或 git history 查看；新架构不新增跨平台抽象。

## 文档索引

| 文档 | 内容 |
| --- | --- |
| [01-current-architecture.md](./01-current-architecture.md) | 现有代码、功能、模块、依赖、数据契约和平台能力梳理 |
| [02-nsgo-mapping.md](./02-nsgo-mapping.md) | NSGo 实际能力、功能映射、差距分析和目标架构 |
| [03-migration-plan.md](./03-migration-plan.md) | 分阶段迁移步骤、验收标准、测试与回滚点 |
| [04-feature-parity-checklist.md](./04-feature-parity-checklist.md) | 逐项功能对照表，防止迁移遗漏 |
| [05-risk-register.md](./05-risk-register.md) | 技术、数据、平台、发布和产品风险 |
| [06-cleanup-report.md](./06-cleanup-report.md) | 本轮清理范围、保留项和后续删除条件 |
| [07-nsgo-upgrade-log.md](./07-nsgo-upgrade-log.md) | NSGo v0.2.0 与依赖升级记录、新特性评估结论 |
| [08-performance-audit.md](./08-performance-audit.md) | 性能、定时任务与资源生命周期排查、基准及验证边界 |

## 证据来源

- 迁移前实现快照：`src-tauri/src/`、`src/`、`src-tauri/migrations/`、`src-tauri/tauri.conf.json`；当前保留部分于 `legacy/backend/` 与 `legacy/frontend/`。
- NSGo 源码：`github.com/nsgo-dev/nsgo`（用户给出的 `oneo-me/nsgo` 为同一仓库入口）。当前固定正式发布版本 **v0.2.0**（2026-11-19）；早期分析的 commit `70cef6a4ed17f7c8428faabc9bb419ff4223745b`（2026-11-12）仅作历史基线。
- NSGo 关键文档：`CHANGELOG.md`、`ARCHITECTURE.md`、`docs/architecture/application.md`、`docs/architecture/services.md`、`docs/architecture/updates.md`、`docs/architecture/tool.md`。

## 重要提醒

NSGo 已发布正式稳定版本（v0.1.0 为首个 release，当前 v0.2.0），不再使用 preview/beta 或伪版本。但 0.x 阶段 API 仍可能变动：迁移方案固定到稳定 tag，并为 AppKit 桥接层保留可替换边界，不把业务代码直接耦合到 NSGo 内部实现。每次升级 NSGo 或其他依赖前，按 `AGENTS.md` §4.1 先评估官方新 API/新特性，合适时优先用官方实现替换重复的自研代码（评估记录见 [07-nsgo-upgrade-log.md](./07-nsgo-upgrade-log.md)）。

## 路径说明

01、02、04 中的 `src-tauri/src/...` 和 `src/...` 是迁移前的代码路径。当前仓库中，剪贴板核心位于 `legacy/backend/src/...`，前端 UI 位于 `legacy/frontend/src/...`；未保留的模块用于描述旧行为，不代表仓库里仍有实现。保留清单见 `legacy/README.md`。
