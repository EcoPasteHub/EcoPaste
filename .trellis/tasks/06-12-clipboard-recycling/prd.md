# Clipboard Recycling

## Goal

为 EcoPaste 增加回收站能力，让删除历史记录默认可恢复，并把彻底删除、自动清理、备份导入和未来同步分阶段落地。

## Design Source

完整需求分析和方案设计保存在 `research/recycling-design.md`。实施任一子任务前先阅读该研究文件，并按当前代码状态重新核对查询、资源清理和事件流。

## Recommended Architecture

- 采用“主表软删除 + 回收状态字段”，不把记录搬到独立回收表。
- 删除 active 历史时切到 recycled 状态，不立即删除数据库行和资源文件。
- 主列表、预览、写回、搜索、统计默认只读取 active 数据。
- 回收站只读取 recycled 数据，支持恢复、批量恢复、彻底删除、清空回收站。
- 自动清理分两段：active 过期或超量时进入回收站；recycled 到期或触发空间压力时才彻底清理。
- FTS 主搜索索引只包含 active 数据。

## Child Tasks

- `clipboard-recycling-foundation`：基础回收站闭环，包含 schema、active-only 查询、恢复、彻底删除、基础 UI 和事件同步。
- `clipboard-recycling-cleanup-storage`：自动清理与存储优化，包含回收站保留策略、空间统计、checkpoint、资源缓存安全清理和批量操作。
- `clipboard-recycling-advanced-integration`：同步、备份、AI 与高级策略，包含生命周期事件、backup/import 语义、派生索引排除和敏感内容策略。

## Acceptance Criteria

- 子任务覆盖原长篇回收站设计的 Phase 1 / 2 / 3 路线图。
- 父任务只承载整体设计和拆分关系，不直接实施代码。
- 子任务实施时不需要回到根目录 TODO 文件。
