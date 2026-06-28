# Clipboard Recycling Advanced Integration

## Goal

让回收机制兼容未来多设备同步、备份导入、AI / embedding 派生能力和更高级的数据生命周期策略。

## Scope

- 设计生命周期事件或 tombstone 表，用于未来同步。
- 为 recycle、restore、purge 增加状态版本或操作时间。
- 备份导出增加“包含回收站”选项。
- 备份 manifest 记录是否包含 recycled。
- 导入策略支持 active-only 和 full-state 两种模式。
- AI、embedding、摘要等派生索引默认排除 recycled。
- 标签系统、收藏系统与回收状态联动规则固化。
- 敏感内容回收站保留期独立配置。
- 评估通用 asset 表，统一图片、未来附件和派生资源引用。

## Implementation Notes

- 参考父任务 `research/recycling-design.md` 的 Phase 3、备份导入、隐私风险、同步冲突和未来通用附件层章节。
- 普通历史备份默认只包含 active，避免用户删除过的内容在新设备重新出现。
- 完整状态备份可通过高级选项包含回收站。
- AI 和搜索默认不使用回收站内容。
- purge 前如果未来有同步，应保留足够同步 grace period 或 tombstone。

## Acceptance Criteria

- 同步设计能表达删除、恢复、彻底删除，不靠猜测最终状态。
- 备份不会默认把用户删除过的内容带到新环境，除非用户显式选择。
- AI 和搜索不会默认使用回收站内容。
- 后续新增附件类型不需要重写回收站主流程。
