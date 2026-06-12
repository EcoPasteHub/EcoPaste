# Clipboard Recycling Cleanup and Storage

## Goal

让回收站真正解决长期空间增长问题，补齐自动清理、存储统计、资源缓存安全清理和批量操作。

## Scope

- 新增回收站保留时长设置：7 天、30 天、90 天、自定义、永不。
- 新增回收站最大条数设置。
- 新增回收站空间上限或本地数据目录空间压力策略。
- 后台 cleanup 改为两段式：active 到 recycled，recycled 到 purge。
- 存储统计区分 active、recycled、可释放空间。
- 清空回收站后执行 WAL checkpoint。
- 设计可选数据库压缩入口。
- recycled 图片缩略图支持空间压力下清理并懒生成。
- `clean_resource_cache` 引用扫描覆盖 recycled 行。
- 支持批量选择、批量恢复、批量彻底删除。
- 回收站支持按类型、大小、删除时间筛选。

## Implementation Notes

- 参考父任务 `research/recycling-design.md` 的 Phase 2、自动清理策略、存储优化策略和风险分析章节。
- active 历史策略只负责移动到 recycled；recycled 策略才负责永久删除。
- 不自动永久删除 active 数据来满足空间上限。
- 图片 purge 前必须扫描 active 和 recycled 引用。
- SQLite 文件大小不会因删除立即缩小；数据页文案需要区分可释放空间和实际文件大小。

## Acceptance Criteria

- 到期 recycled 自动彻底删除。
- 超出 recycled 最大条数时永久删除最早进入回收站的记录。
- 清空回收站后图片资源按引用安全释放。
- 数据页能解释为什么删除后空间未立即下降，以及清空回收站可释放多少。
- 资源缓存清理不会误删回收站仍引用的图片。
