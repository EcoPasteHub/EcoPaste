# Clipboard Recycling Foundation

## Goal

落地基础回收站闭环：用户删除 active 条目后可恢复，并且不破坏现有主列表、搜索、预览、写回和资源引用。

## Scope

- 为 `clipboard_items` 设计并落地生命周期字段。
- 所有主列表查询默认过滤 active。
- 写回、粘贴、预览、收藏、置顶、备注默认只作用 active。
- 修改 `upsert_item` 为状态感知去重：active 优先，recycled 命中则自动恢复。
- 删除单条从硬删除改为移入回收站。
- 清空历史从硬删除改为批量移入回收站。
- 新增查询 recycled 列表的 Rust 命令。
- 新增恢复单条、批量恢复命令。
- 新增彻底删除单条、清空回收站命令。
- FTS 改为 active-only。
- 图片彻底删除前增加引用检查。
- 前端增加回收站入口和基础列表。
- 删除 toast 增加 Undo。
- 回收站支持恢复、彻底删除、清空。
- 更新 i18n 文案。

## Implementation Notes

- 参考父任务 `research/recycling-design.md` 的 Phase 1、数据库、FTS、命令边界和 UI/UX 章节。
- 回收、恢复、purge 都应 emit 明确事件，让主列表、回收站、预览、Footer 统一同步。
- 删除时不应修改 `updated_at`；恢复时也保持原最近使用排序。
- 收藏和置顶保护沿用现有删除设置，但语义改为“移入回收站”。
- 当前仍处于未发版开发期时，可直接合并 schema 到 `0001_init.sql`；若实施时版本已经发布，则新增 migration。
- 如果用户可感知行为相对旧版新增，实施时更新 `CHANGELOG.md`。

## Acceptance Criteria

- 删除 active 条目后主列表消失，回收站出现。
- Undo 可恢复。
- 回收站恢复后主列表可见，内容、备注、收藏状态保留。
- 回收站彻底删除图片后，只有无其它引用时才删除原图和缩略图。
- 主搜索不命中回收站内容。
- 收藏和置顶保护仍生效。
- 增加 Rust 仓储层测试并完成前端主路径验证。
