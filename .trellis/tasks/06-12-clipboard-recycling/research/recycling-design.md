# Clipboard Recycling 规划 TODO

> 本文件仅做需求分析与方案设计，不包含实现代码。
> 当前任务禁止修改源码、配置、数据库结构、测试、构建脚本和既有文档；真正实施时再按本路线图拆任务。

## 0. 设计结论

推荐采用“主表软删除 + 回收状态字段”的方案，而不是把记录搬到独立回收表。

核心原则：

- 删除历史记录时，默认从 active 状态切到 recycled 状态，不立即删除数据库行和资源文件。
- 主列表、预览、写回、搜索、统计默认只读取 active 数据。
- 回收站只读取 recycled 数据，支持恢复、批量恢复、彻底删除、清空回收站。
- 自动历史清理分两段：active 过期或超量时进入回收站；recycled 到期或触发空间压力时才彻底清理。
- 图片原图在 recycled 阶段保留；缩略图可保留或作为可再生缓存提前清理；彻底删除时必须做引用检查后再删资源。
- FTS 主搜索索引只包含 active 数据；recycled 数据不参与主搜索、主统计和默认 AI/同步派生索引。

推荐新增概念：

- `active`：正常历史记录，可搜索、可写回、可粘贴、可统计。
- `recycled`：回收站记录，只在回收站页可见，可恢复或彻底删除。
- `purged`：彻底删除后的终态，不再保留行；未来同步需要 tombstone 记录时再单独设计。

## 1. 当前架构分析

### 1.1 数据库结构

当前 SQLite 数据库由 Rust `sqlx` 管理，数据库文件为 app data 下的 `db/clipboard.db`，启用 WAL、foreign keys 和 migration。

现有核心表：

- `clipboard_items`：剪贴板历史主表。
- `clipboard_groups`：用户分组。
- `clipboard_apps`：复制来源应用，主键是 macOS bundle id 或 Windows exe 绝对路径。
- `file_type_icons`：文件类型图标缓存索引。
- `clipboard_items_fts`：FTS5 外部内容虚表，索引 `search_text` 与 `note`。

`clipboard_items` 当前字段承载了历史记录的全部核心状态：

- 身份与类型：`id`、`kind`、`sub_kind`、`platform`。
- 关系：`group_id`、`source_app_id`。
- 内容：`content`、`content_hash`、`search_text`、`summary`、`file_types`。
- 大小与媒体信息：`size`、`width`、`height`。
- 用户元数据：`use_count`、`is_favorite`、`is_pinned`、`is_sensitive`、`note`。
- 时间：`created_at`、`updated_at`。

当前没有以下字段：

- 删除状态。
- 回收时间。
- 计划彻底清理时间。
- 删除来源或删除原因。
- 是否从回收站恢复过。

当前 `updated_at` 的语义是“内容被重新使用”的时间，不是任意元数据修改时间。回收机制不能用 `updated_at` 记录删除或恢复动作，否则会污染 most-recent-use 排序。

### 1.2 存储结构

当前本地数据目录大致拆为：

- `db/`：SQLite 主文件与 WAL/SHM sidecar。
- `resources/clipboard-images/origin/<hash-prefix>/<hash>.png`：剪贴板图片原图。
- `resources/clipboard-images/thumbnails/<hash-prefix>/<hash>.png`：图片缩略图，列表或预览时懒生成。
- `resources/app-icons/`：来源应用图标缓存。
- `resources/file-icons/`：文件类型图标缓存。
- `config/settings.json`：设置。
- `state/`：窗口状态等运行状态。

图片类型记录：

- `kind = image`。
- `content` 存图片文件名，例如 `<hash>.png`。
- 原图和缩略图路径由 `ImageStore` 根据文件名和 hash 分片推导。
- 当前删除图片记录时，会直接删除原图和缩略图。

Files 类型记录：

- `kind = files`。
- `content` 存换行分隔的本地路径列表。
- `file_types` 用紧凑字符串记录每个路径是文件还是目录。
- EcoPaste 不复制原始外部文件本体，只保存路径和图标缓存。
- 路径对应的外部文件删除与否不由 EcoPaste 控制，列表和预览会实时检测 `exists`。

文本、HTML、RTF：

- 都作为 `kind = text` 入库。
- HTML/RTF 的 `content` 存源表示。
- `search_text` 存 OS 同时提供的纯文本表示。
- `summary` 存列表摘要。
- 大文本、HTML、RTF 都在 SQLite 内部占用数据库空间。

### 1.3 搜索结构

当前 FTS5：

- `clipboard_items_fts` 使用外部内容表 `clipboard_items`。
- 触发器在 `INSERT`、`UPDATE`、`DELETE` 时同步 FTS。
- 主列表搜索中，3 个及以上字符走 FTS，1 到 2 个字符走 `LIKE` 兜底。
- 搜索范围为 `search_text` 和 `note`。

目前因为没有回收状态，所有主表记录都会参与 FTS。引入回收站后，如果不改触发器，recycled 记录仍会占用 FTS 索引，并可能被主搜索命中，只能靠外层查询过滤。这会保留索引体积，不能解决搜索性能和索引膨胀问题。

### 1.4 数据生命周期

当前生命周期：

1. OS 剪贴板监听或手动读取。
2. Rust 读取 payload。
3. Rust 识别类型、落盘图片、生成摘要和搜索字段。
4. `upsert_item` 按 `content_hash` 去重。
5. 新内容插入主表；重复内容刷新 `use_count` 与 `updated_at`。
6. 前端收到 `clipboard://updated` 后刷新列表。
7. 用户删除、清空历史或后台 cleanup 时，数据库行被直接 `DELETE`。
8. 图片行删除后，调用 `ImageStore::remove` 删除原图和缩略图。
9. 资源清理命令通过“数据库仍引用的文件名”保留图片、应用图标和文件图标，其余删除。

现状是“硬删除”模型，不存在恢复窗口。

## 2. 问题分析

### 2.1 用户删除不可恢复

当前单条删除、清空历史、自动历史清理都会永久删除记录。误删图片、HTML、RTF、大段文本后无法恢复。

### 2.2 自动清理过于破坏性

现有后台清理按保留时长和最大条数直接删除非收藏、非置顶记录。它能控制增长，但没有“先回收、再清空”的缓冲区。

### 2.3 图片资源删除假设过强

当前图片删除逻辑假设“同图至多一行”，删除图片行后文件必为孤儿。但回收机制会引入以下情况：

- active 行和 recycled 行可能引用同一图片文件。
- 用户删除图片后又复制同一张图片，可能恢复旧行或产生新的 active 引用。
- 彻底删除 recycled 行时，必须确认没有任何 active 或 recycled 行仍引用该图片文件。

因此后续不能再单纯依赖“删除行即删文件”的假设。

### 2.4 FTS 与软删除存在天然冲突

如果只加 `is_deleted` 或 `deleted_at`，但不调整 FTS 触发器：

- 主搜索需要每次额外过滤 active。
- FTS 索引仍包含 recycled 内容，索引体积不会下降。
- 大 HTML、RTF、大文本进入回收站后仍会影响索引维护成本。

因此回收状态必须和 FTS 策略一起设计。

### 2.5 SQLite 文件不会立即变小

即使彻底删除大文本或大量 HTML/RTF，SQLite 文件通常也只是释放内部 page，不一定立刻缩小磁盘文件。WAL 模式下还需要考虑 checkpoint、WAL 文件大小和 VACUUM 策略。

### 2.6 清空历史与清空回收站语义需要拆开

当前“清空记录”可以理解为永久删除。引入回收站后应拆成：

- 清空历史：把 active 历史移入回收站。
- 清空回收站：彻底删除 recycled 历史及可安全删除的资源。

如果 UI 文案不拆清，会导致用户误判存储空间为何没有立刻下降。

### 2.7 备份、导入、资源清理口径会变化

当前备份会复制整个数据库与 resources。引入回收站后需要决定：

- 普通历史备份是否包含回收站。
- 导入时是否恢复回收站状态。
- 资源清理判断引用时是否把 recycled 行算作引用。

## 3. 回收机制设计

### 3.1 总体架构

推荐在 `clipboard_items` 主表增加生命周期字段，而不是创建独立回收表。

推荐理由：

- 保留原 `id`，恢复时不需要重建关系。
- 保留 `group_id`、`source_app_id`、收藏、置顶、备注、敏感标记和统计字段。
- 减少在主表和回收表之间搬运大字段的成本。
- FTS、备份、资源清理都可以围绕同一主表做状态过滤。
- 未来同步时可以把状态变化当作同一条记录的生命周期事件。

不推荐独立回收表的原因：

- 需要复制 `clipboard_items` 的几乎所有字段，schema 会双份演进。
- 恢复需要跨表搬回，容易破坏 rowid、FTS 和关联关系。
- 资源引用要同时扫描两张表。
- 备份、导入、搜索、统计都要双路径维护。

推荐状态字段：

- `lifecycle_state`：`active` 或 `recycled`。
- `recycled_at`：进入回收站的时间。
- `recycled_reason`：进入回收站的原因，例如 `user_delete`、`clear_history`、`retention`、`max_count`、`storage_quota`。
- `purge_after`：最早允许自动彻底删除的时间，手动彻底删除不受此字段限制。

可选字段：

- `restored_at`：最近一次从回收站手动恢复的时间。仅用于审计或 UI，不参与排序。
- `recycled_batch_id`：批量删除或清空历史时的批次 id，便于 Undo 整批恢复。

### 3.2 命令边界

推荐新增或调整命令语义：

- 删除单条：把 active 条目移动到 recycled。
- 批量删除：把 active 条目批量移动到 recycled。
- 恢复单条：把 recycled 条目恢复到 active。
- 批量恢复：把多个 recycled 条目恢复到 active。
- 彻底删除单条：从 recycled 中永久删除。
- 批量彻底删除：永久删除多个 recycled 条目。
- 清空历史：把符合条件的 active 条目移动到 recycled。
- 清空回收站：永久删除全部或筛选后的 recycled 条目。
- 查询历史：默认只查 active。
- 查询回收站：只查 recycled，按 `recycled_at DESC` 排序。
- 读取存储统计：区分总占用、active 引用、recycled 可回收占用。

命令层仍保持薄：参数校验、调用 db/clipboard/storage 下层模块、emit 事件。

### 3.3 事件设计

现有 `clipboard://updated` payload 主要覆盖新增、去重、清理、导入。回收机制应扩展事件语义，避免列表、回收站、预览、Footer 统计不同步。

建议事件 payload 表达：

- `recycled`：移动到回收站的数量和 ids。
- `restored`：恢复数量和 ids。
- `purged`：彻底删除数量和 ids。
- `trashEmptied`：清空回收站数量。
- `cleanup`：后台任务影响数量，但需区分 moved-to-recycle 和 purged。
- `imported`：导入后全量刷新。

主列表响应：

- 新增 active：现有逻辑。
- recycled：从当前 active 列表移除。
- restored：若当前筛选命中，可刷新或累积新记录 badge。
- purged：如果预览窗口正打开该记录，应关闭。

回收站视图响应：

- recycled：插入或刷新回收站列表。
- restored：从回收站列表移除。
- purged：从回收站列表移除。
- trashEmptied：清空本地列表。

### 3.4 去重规则

引入 recycled 后，`content_hash` 去重必须带状态意识。

推荐规则：

1. 入库时优先查 active 中同 `content_hash` 的记录。
2. 如果 active 命中，沿用当前去重逻辑，刷新 `use_count` 与 `updated_at`。
3. 如果 active 未命中，但 recycled 命中，推荐将 recycled 记录自动恢复为 active，并按“重新复制同一内容”处理：
   - 清空 `recycled_at`、`recycled_reason`、`purge_after`。
   - 刷新 `use_count` 与 `updated_at`。
   - 更新 `source_app_id` 为本次复制来源。
   - 保留 `note`、`is_favorite`、`is_pinned`、`group_id` 等用户元数据。
4. 如果 active 和 recycled 都未命中，插入新 active 行。

这样可以避免同一图片文件被多行引用导致的复杂引用计数，也符合“从回收站重新出现”的用户直觉。

需要产品确认的细节：

- 自动恢复 recycled 记录时，是否保留 `is_pinned`。保留会让重新复制的旧置顶项直接回到顶部；清除会更接近“重新加入历史”。建议 Phase 1 先保留，避免丢用户元数据。

### 3.5 收藏与置顶保护

现有设置已经有：

- 是否允许删除收藏条目。
- 删除收藏条目前是否确认。
- 是否允许删除置顶条目。
- 删除置顶条目前是否确认。
- 收藏条目是否只能在收藏分组删除。

回收机制应沿用这些保护，但“删除”动作语义改为“移入回收站”。彻底删除时应再次确认，且对收藏/置顶项使用更强的危险提示。

自动回收仍应默认跳过收藏和置顶条目，延续现有 cleanup 行为。

## 4. 数据生命周期设计

### 4.1 创建

来源：

- OS 监听。
- 手动重新读取。
- 未来导入或同步。

流程：

1. 读取剪贴板 payload。
2. 根据设置过滤内容类型与大小。
3. 图片原图落盘。
4. 构造 item。
5. 按状态感知去重规则写入或恢复。
6. active 行进入主列表和 active FTS。
7. emit 更新事件。

新建时：

- `lifecycle_state = active`。
- `recycled_at = NULL`。
- `recycled_reason = NULL`。
- `purge_after = NULL`。

### 4.2 普通删除

用户删除单条 active 记录：

1. 前端按现有设置判断是否允许删除收藏或置顶项。
2. 如需确认，弹出确认框。
3. Rust 将记录标记为 recycled。
4. 不删除图片原图。
5. active FTS 删除该行索引。
6. emit recycled 事件。
7. 前端从主列表移除，并显示“已移至回收站”的 toast。
8. toast 提供撤销入口，撤销调用 restore。

删除时不应修改 `updated_at`。

### 4.3 批量删除

批量删除与单条删除一致，但建议记录同一个 `recycled_batch_id`，用于一次 Undo 恢复整批。

批量删除结果应返回：

- 成功移入回收站数量。
- 被保护跳过数量。
- 不存在或状态不匹配数量。

### 4.4 清空历史

“清空历史”应从永久删除改为批量移入回收站。

推荐规则：

- 默认保留收藏和置顶，沿用现有确认弹窗选项。
- 用户显式勾选删除收藏或置顶时，也只是移入回收站。
- 成功后主列表清空或刷新。
- toast 文案使用“已移入回收站”，不要说“已删除”。

### 4.5 恢复

用户从回收站恢复：

1. 校验记录当前是 recycled。
2. 将状态改回 active。
3. 清空 `recycled_at`、`recycled_reason`、`purge_after`。
4. 重新加入 active FTS。
5. 不修改 `updated_at`，保持它原本的最近使用排序。
6. emit restored 事件。

如果原分组被删除，`group_id` 会因外键规则变成 NULL，恢复后进入默认“全部”。

### 4.6 复制或粘贴回收站记录

推荐 Phase 1 不允许直接从回收站记录执行粘贴或写回，避免“已删除内容仍可被意外使用”的认知混乱。

回收站卡片动作：

- 恢复。
- 彻底删除。
- 预览。

如果后续需要“从回收站复制”，应在 UI 上明确这是临时访问，且不改变 active 状态，或先恢复再复制。

### 4.7 彻底删除

彻底删除仅作用于 recycled 记录。

流程：

1. 删除主表行。
2. FTS 中不应再有该行索引；若 active-only FTS 正确实现，recycled 行本来不在 FTS。
3. 对图片资源做引用检查：
   - 若没有任何 active 或 recycled 行引用该 `content` 文件名，删除 origin 和 thumbnail。
   - 若仍有引用，保留文件。
4. 对文件图标、应用图标不立即逐项删除，交由资源缓存清理统一处理。
5. emit purged 事件。

### 4.8 自动历史回收

现有 cleanup 后台任务应改成：

- active retention 或 max count 命中时：移动到 recycled。
- recycled retention、数量上限或空间压力命中时：永久 purge。

启动时仍可运行一次 cleanup，但必须尊重两段式语义。

### 4.9 数据目录迁移与备份

数据目录迁移是完整搬迁本地状态，应该包含 active、recycled、数据库和 resources。

历史备份需要产品决策：

- 默认建议只导出 active 历史，避免用户删除过的内容在另一台设备导入后重新出现。
- 提供“包含回收站”选项，用于完整本地状态备份。
- overwrite 导入如果是完整备份，应保留回收站状态。
- merge 导入默认只合并 active；如果导入包包含回收站且用户选择包含，应按状态合并。

## 5. 自动清理策略

### 5.1 策略分层

建议把自动策略拆成两组：

active 历史策略：

- 历史保留时长。
- 历史最大条数。
- 是否跳过收藏。
- 是否跳过置顶。
- 可选：按内容类型设置保留策略。

recycled 回收站策略：

- 回收站保留时长。
- 回收站最大条数。
- 回收站最大空间。
- 空间压力下的清理优先级。
- 是否对敏感内容使用更短保留期。

### 5.2 时间策略

应支持：

- 7 天自动清理回收站。
- 30 天自动清理回收站。
- 90 天自动清理回收站。
- 自定义天数。
- 永不自动清理。

推荐默认值：

- active 历史沿用当前默认：永久保留，最大条数不限。
- 回收站默认 30 天后自动彻底清理。

理由：

- 保持现有历史保留行为不突然改变。
- 回收站提供恢复窗口，同时能最终释放大图片和大文本。

时间口径：

- active 历史过期：Phase 1 可沿用当前 `created_at` 口径，避免行为突变。
- 后续可评估切换为 `updated_at` 或提供“按创建时间/最近使用时间”选项。
- 回收站过期：一律按 `recycled_at`。

### 5.3 数量策略

active 最大条数：

- 超出部分从 active 移入回收站。
- 排序口径沿用当前 cleanup：非收藏、非置顶集合按创建时间保留最新。
- 后续可评估按 `updated_at` 保留最近使用项。

recycled 最大条数：

- 超出部分永久删除最早进入回收站的记录。
- 排序口径为 `recycled_at ASC`。
- 永远不影响 active 数据。

### 5.4 空间策略

建议新增可选空间上限：

- 回收站最大占用。
- 本地数据目录目标上限。
- 图片资源目标上限。

空间压力清理顺序：

1. 已过 `purge_after` 的 recycled 项。
2. recycled 中最早删除的图片和大文本。
3. recycled 中其它大体积内容。
4. 可再生缩略图缓存。
5. 文件图标缓存中不再引用的文件。
6. 只提示用户，不自动删除 active 数据。

不建议自动永久删除 active 数据来满足空间上限。active 数据只能被 active 历史策略移动到回收站。

### 5.5 内容类型策略

建议 Phase 2 支持按类型清理：

- 图片：可设置更短回收站保留期或空间上限。
- HTML/RTF：可设置更短回收站保留期，因为它们直接占 DB 和 FTS 维护成本。
- 大文本：按 size 阈值进入“大内容”清理优先级。
- Files：只保存路径，通常空间成本低，不需要优先清理。

敏感内容策略：

- `is_sensitive = true` 的记录进入回收站后，建议默认 7 天后彻底清理，或提供“敏感内容删除后立即彻底删除”的隐私选项。
- 如果启用敏感内容更短保留期，UI 必须清楚说明。

### 5.6 调度策略

现有 `cleanup_interval_hours` 可继续作为后台检查周期。

建议：

- 启动时运行一次 cleanup。
- 周期到达时运行 cleanup。
- 设置变更后无需重启，下一次周期读取最新设置。
- 手动“清空回收站”立即运行 purge。
- 手动“清理资源缓存”只清孤儿缓存，不改变 active/recycled 状态。

## 6. 存储优化策略

### 6.1 图片

recycled 阶段：

- 保留 origin 原图，保证恢复后可正常预览和写回。
- 缩略图可以保留，提升回收站浏览速度。
- 如果用户启用空间优化，可先删除 recycled 图片的 thumbnail，恢复或预览时再懒生成。

purge 阶段：

- 删除 DB 行。
- 查询是否仍有任何 active 或 recycled 图片行引用同一 `content` 文件名。
- 无引用时删除 origin 和 thumbnail。
- 删除空分片目录。

### 6.2 Files 类型

Files 记录不保存外部文件本体，只保存路径列表和类型信息。

策略：

- 进入回收站不影响原始文件。
- 恢复只恢复路径记录。
- 回收站预览仍可显示路径是否存在。
- 彻底删除只删除数据库行。
- 文件图标缓存不随单条记录删除，交由缓存清理统一处理。

### 6.3 HTML、RTF、大文本

这些内容直接存储在 SQLite `content` 字段中，进入回收站不会释放数据库空间。

优化策略：

- 回收时从 active FTS 移除，降低主搜索索引体积和搜索干扰。
- purge 后删除行，释放 SQLite 内部页面。
- 当一次 purge 删除大量大内容后，执行 WAL checkpoint。
- 提供“压缩数据库”或自动 VACUUM 策略，但不要每次 purge 都 VACUUM，避免卡顿。

VACUUM 建议：

- Phase 1 不自动 VACUUM，只做 checkpoint。
- Phase 2 在清空回收站后，如果估算可回收空间超过阈值，提示用户或后台低频执行。
- 执行 VACUUM 前应暂停剪贴板写入或确保命令串行，避免数据库锁冲突。

### 6.4 缩略图与派生缓存

缩略图、应用图标、文件图标属于派生缓存或共享资源。

建议：

- 缩略图：可按引用状态和空间压力清理；缺失后可再生成。
- 应用图标：由 `clipboard_apps` 引用，不应因条目进入回收站删除。
- 文件图标：由 `file_type_icons` 引用，不应因单条 files 记录删除立即删除。
- `clean_resource_cache` 引用扫描必须包含 recycled 行，否则回收站图片会被误删。

### 6.5 未来通用附件层

如果未来支持更多附件类型，建议引入通用 asset 概念：

- `clipboard_assets`：记录 asset id、类型、文件名、字节数、hash、created_at。
- `clipboard_item_assets`：记录 item 与 asset 的引用关系。

Phase 1 不建议引入，当前图片文件名引用足够。等同步、AI、附件增多时再抽象。

## 7. UI/UX 建议

### 7.1 回收站入口

推荐两个入口：

- 主窗口分组区或更多菜单中提供“回收站”入口。
- 偏好设置的数据页提供“打开回收站”和“清空回收站”入口。

主窗口是高频操作区，回收站入口应可达但不抢占“全部/文本/图片/文件/收藏”的主流程。可放在 group tab 的尾部或 header 的更多菜单中。

### 7.2 回收站列表

回收站列表建议独立视图，不混在普通历史列表中。

展示字段：

- 原内容卡片。
- 删除时间。
- 删除原因。
- 原类型。
- 大小。
- 来源应用。
- 收藏/置顶/备注标记可展示但不可直接编辑，Phase 1 可先只展示。

排序：

- 默认 `recycled_at DESC`，最近删除在前。
- 可选按大小、类型排序，Phase 2 再做。

搜索：

- Phase 1 可支持简单 keyword 过滤。
- 不参与主历史搜索。

### 7.3 删除交互

普通删除：

- 文案改为“移至回收站”。
- 成功 toast：`已移至回收站`。
- Toast 操作：`撤销`。
- 收藏/置顶保护沿用现有设置。

彻底删除：

- 仅在回收站中出现。
- 使用 danger 样式。
- 二次确认文案明确“此操作无法撤销”。

清空历史：

- 文案建议改为“清空历史”或“全部移至回收站”。
- 确认框说明不会立即释放全部空间，释放空间请清空回收站。

清空回收站：

- 独立危险操作。
- 确认框说明会删除图片原图、缩略图和数据库内容，无法恢复。

### 7.4 恢复交互

单条恢复：

- 回收站卡片提供恢复按钮。
- 成功 toast：`已恢复`，可提供“查看”。
- 恢复后记录回到原分组；如果原分组不存在，则在全部中可见。

批量恢复：

- 支持多选。
- 顶部批量操作栏提供“恢复”和“彻底删除”。
- 恢复后从回收站列表移除。

Undo：

- 普通删除后的 toast 撤销等价于 restore。
- 批量删除后的 toast 撤销按 `recycled_batch_id` 批量恢复。
- 即使 toast 消失，用户仍可从回收站恢复。

### 7.5 空状态与统计

主列表空状态：

- 不应提示“没有历史”如果回收站里还有记录；可在空状态下提供“查看回收站”。

回收站空状态：

- 文案说明删除的记录会先来到这里。

偏好数据页：

- 展示本地数据总占用。
- 展示回收站占用或预计可释放空间。
- 展示图片资源、数据库、设置占用。

### 7.6 快捷键

主列表：

- 现有删除快捷键保持，但语义改成移至回收站。

回收站：

- Delete 或 Backspace：彻底删除，必须确认。
- Enter：恢复。
- Cmd/Ctrl+A：全选。

Windows 主窗口仍需遵守现有约束：涉及 UI 的键盘处理不能只依赖 Web `keydown`，主窗口场景需要 Rust `keyboard://nav` 或既有键盘钩子协同。

## 8. 数据库改造建议

> 本节只提出方案，不实施。

### 8.1 主表字段

建议在 `clipboard_items` 增加：

- `lifecycle_state TEXT NOT NULL DEFAULT 'active'`。
- `recycled_at TEXT`。
- `recycled_reason TEXT`。
- `purge_after TEXT`。
- 可选：`restored_at TEXT`。
- 可选：`recycled_batch_id TEXT`。

字段约束：

- `lifecycle_state` 仅允许 `active`、`recycled`。
- `recycled_at` 为 NULL 时必须是 active。
- recycled 记录应有 `recycled_at`。
- `updated_at` 不用于记录删除、恢复、备注、收藏、置顶变化。

### 8.2 索引

建议新增索引：

- `(lifecycle_state, updated_at, created_at)`：主列表默认排序。
- `(lifecycle_state, recycled_at)`：回收站列表排序。
- `(lifecycle_state, kind)`：类型分组过滤。
- `(lifecycle_state, is_favorite)`：收藏过滤。
- `(lifecycle_state, is_pinned)`：置顶过滤。
- `(lifecycle_state, content_hash)`：状态感知去重。
- `(purge_after)`：自动清空回收站。

当前 `idx_clipboard_items_content_hash` 需要评估是否替换为状态组合索引，避免 recycled 命中污染 active 去重。

### 8.3 查询层

建议增加查询参数：

- `lifecycleState`：默认为 active。
- 回收站命令显式传 recycled。

所有现有查询必须默认过滤 `active`：

- 列表。
- 单条列表视图查询。
- 完整预览查询。
- 写回剪贴板。
- 粘贴。
- 收藏/置顶/备注修改。
- 来源应用引用判断。
- 统计。

回收站专用命令可以读取 recycled。

### 8.4 FTS

推荐把主 FTS 改为 active-only。

触发器策略：

- 插入 active 行时写入 FTS。
- 插入 recycled 行时不写入 FTS。
- active 更新内容或备注时更新 FTS。
- active 转 recycled 时从 FTS 删除。
- recycled 转 active 时重新写入 FTS。
- recycled 内部只改 `recycled_at` 等回收字段时不写入 FTS。

回收站搜索策略：

- Phase 1：使用 `LIKE` 搜索 recycled 的 `search_text` 与 `note`。
- Phase 2：如果回收站搜索量大，再考虑独立 `clipboard_recycled_fts`。

### 8.5 资源引用查询

图片永久删除前必须查询所有未 purged 记录：

- active image 行。
- recycled image 行。

`clean_resource_cache` 的 referenced image files 也必须包含 active 和 recycled 行。

如果后续引入 asset 表，则以 asset 引用关系为准。

### 8.6 设置模型

建议新增设置结构：

- `clipboard.recycle.enabled`：是否启用回收站。建议默认 true。
- `clipboard.recycle.retention`：回收站保留时长。
- `clipboard.recycle.maxCount`：回收站最大条数，0 表示不限。
- `clipboard.recycle.maxBytes`：回收站最大占用，0 表示不限。
- `clipboard.recycle.cleanupIntervalHours`：可复用现有 cleanup 周期，也可独立设置。
- `clipboard.recycle.sensitiveRetention`：敏感内容回收站保留时长，可选。
- `clipboard.recycle.thumbnailPolicy`：保留缩略图或空间压力下可删除。

也可以先不做独立 cleanup 周期，直接复用 `clipboard.history.cleanup_interval_hours`。

### 8.7 命令常量与事件常量

前端命令名需要加入 `src/constants/commands.ts`。

事件名仍建议使用现有 `clipboard://updated`，但 payload 明确区分新增、回收、恢复、彻底删除。若后续事件过于复杂，再拆：

- `clipboard://recycled`。
- `clipboard://restored`。
- `clipboard://purged`。

Phase 1 推荐先复用 `clipboard://updated`，减少监听分散。

### 8.8 当前版本下的 migration 策略

当前 `package.json` 版本仍是 `0.6.0-beta.3`，按项目约定仍属于未发版开发阶段。真正实施 schema 变更时，可以直接合并到 `0001_init.sql` 和相关测试，不需要新增兼容迁移。

如果实施时版本已经变更为发布版本，则必须新增 migration，并为旧用户数据补状态默认值。

## 9. 风险分析

### 9.1 数据膨胀风险

回收站会延长数据生命周期。短期内总占用可能比当前硬删除更大，尤其是图片、HTML、RTF 和大文本。

缓解：

- 默认回收站 30 天自动清理。
- 展示回收站占用与可释放空间。
- 支持手动清空回收站。
- 支持空间压力策略。

### 9.2 搜索性能风险

如果 recycled 数据仍保留在主 FTS，索引体积不会下降。

缓解：

- active-only FTS。
- recycled 搜索先用 LIKE。
- 大量 purge 后做 checkpoint 和可选 VACUUM。

### 9.3 图片误删风险

同一图片文件可能被 active 和 recycled 状态复用。如果 purge 时不做引用检查，会破坏 active 图片预览和写回。

缓解：

- 删除文件前查询所有状态引用。
- clean cache 引用扫描包含 recycled。
- 去重优先自动恢复 recycled，减少重复引用。

### 9.4 用户认知风险

用户可能认为“删除”后空间应立即释放，但回收站会保留内容。

缓解：

- 文案使用“移至回收站”。
- 清空历史确认框说明不等于释放全部空间。
- 数据页展示“回收站可释放空间”。

### 9.5 隐私风险

敏感内容删除后仍在回收站停留，可能不符合用户预期。

缓解：

- 敏感内容删除时提示或提供立即彻底删除选项。
- 支持敏感内容较短回收站保留期。
- 回收站中继续执行敏感内容脱敏展示。

### 9.6 并发与竞态风险

后台 cleanup、用户恢复、用户清空回收站、剪贴板监听入库可能同时发生。

缓解：

- DB 操作使用事务。
- 状态变更命令带状态条件，例如只恢复 recycled，只回收 active。
- purge 和 restore 冲突时以后提交的状态条件为准，失败返回明确结果。

### 9.7 备份与导入风险

如果备份默认包含回收站，用户在新设备导入后可能看到已删除内容。如果默认不包含，完整灾备能力变弱。

缓解：

- 普通历史备份默认只包含 active。
- 高级选项允许包含回收站。
- manifest 记录是否包含 recycled。

### 9.8 SQLite 文件收缩风险

purge 后磁盘文件可能不变小，用户会认为清理无效。

缓解：

- 数据页区分逻辑删除、可释放、实际文件大小。
- 清空回收站后运行 checkpoint。
- 提供低频数据库压缩能力。

### 9.9 UI 状态同步风险

当前单条删除不广播，前端本地 mutate。回收站引入后，主列表和回收站可能同时打开或有预览窗口，单靠本地 mutate 不够。

缓解：

- 回收、恢复、purge 都 emit 明确事件。
- 主列表、回收站、预览、Footer 统一监听。
- 单条命令返回变更结果，调用方仍可乐观更新。

### 9.10 未来同步冲突风险

多设备同步时，回收、恢复、彻底删除会和新增、去重产生冲突。

缓解：

- 把 recycle/restore/purge 设计成明确生命周期事件。
- 未来添加 `state_version`、`state_updated_at` 或 tombstone 表。
- purge 前保留足够同步 grace period。

## 10. 实施路线图

### Phase 1：基础回收站闭环

目标：让用户删除可恢复，且不破坏现有主列表和搜索。

任务：

- [ ] 为 `clipboard_items` 设计并落地生命周期字段。
- [ ] 所有主列表查询默认过滤 active。
- [ ] 写回、粘贴、预览、收藏、置顶、备注默认只作用 active。
- [ ] 修改 `upsert_item` 为状态感知去重：active 优先，recycled 命中则自动恢复。
- [ ] 删除单条从硬删除改为移入回收站。
- [ ] 清空历史从硬删除改为批量移入回收站。
- [ ] 新增查询 recycled 列表的 Rust 命令。
- [ ] 新增恢复单条、批量恢复命令。
- [ ] 新增彻底删除单条、清空回收站命令。
- [ ] FTS 改为 active-only。
- [ ] 图片彻底删除前增加引用检查。
- [ ] 前端增加回收站入口和基础列表。
- [ ] 删除 toast 增加 Undo。
- [ ] 回收站支持恢复、彻底删除、清空。
- [ ] 更新 i18n 文案。
- [ ] 增加 Rust 仓储层测试和前端主路径验证。
- [ ] 如果用户可感知行为相对旧版新增，实施时更新 `CHANGELOG.md`。

验收：

- 删除 active 条目后主列表消失，回收站出现。
- Undo 可恢复。
- 回收站恢复后主列表可见，内容、备注、收藏状态保留。
- 回收站彻底删除图片后，只有无其它引用时才删除原图和缩略图。
- 主搜索不命中回收站内容。
- 收藏和置顶保护仍生效。

### Phase 2：自动清理与存储优化

目标：让回收站真正解决长期空间增长。

任务：

- [ ] 新增回收站保留时长设置：7 天、30 天、90 天、自定义、永不。
- [ ] 新增回收站最大条数设置。
- [ ] 新增回收站空间上限或本地数据目录空间压力策略。
- [ ] 后台 cleanup 改为两段式：active 到 recycled，recycled 到 purge。
- [ ] 存储统计区分 active、recycled、可释放空间。
- [ ] 清空回收站后执行 WAL checkpoint。
- [ ] 设计可选数据库压缩入口。
- [ ] recycled 图片缩略图支持空间压力下清理并懒生成。
- [ ] `clean_resource_cache` 引用扫描覆盖 recycled 行。
- [ ] 支持批量选择、批量恢复、批量彻底删除。
- [ ] 回收站支持按类型、大小、删除时间筛选。

验收：

- 到期 recycled 自动彻底删除。
- 超出 recycled 最大条数时永久删除最早进入回收站的记录。
- 清空回收站后图片资源按引用安全释放。
- 数据页能解释为什么删除后空间未立即下降，以及清空回收站可释放多少。

### Phase 3：同步、备份、AI 与高级策略

目标：让回收机制兼容未来多设备和派生能力。

任务：

- [ ] 设计生命周期事件或 tombstone 表，用于未来同步。
- [ ] 为 recycle、restore、purge 增加状态版本或操作时间。
- [ ] 备份导出增加“包含回收站”选项。
- [ ] 备份 manifest 记录是否包含 recycled。
- [ ] 导入策略支持 active-only 和 full-state 两种模式。
- [ ] AI/embedding/摘要等派生索引默认排除 recycled。
- [ ] 标签系统、收藏系统与回收状态联动规则固化。
- [ ] 敏感内容回收站保留期独立配置。
- [ ] 评估通用 asset 表，统一图片、未来附件和派生资源引用。

验收：

- 同步设计能表达删除、恢复、彻底删除，不靠猜测最终状态。
- 备份不会默认把用户删除过的内容带到新环境，除非用户显式选择。
- AI 和搜索不会默认使用回收站内容。
- 后续新增附件类型不需要重写回收站主流程。

