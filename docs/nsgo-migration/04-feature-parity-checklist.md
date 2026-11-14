# 04 · 功能 Parity 清单

> 用途：迁移期间逐项勾对，防止“代码搬完了但功能少了一块”。
> 状态说明：`待迁移`、`迁移中`、`已通过`、`参考`（只保留作为旧实现参考）、`不纳入 macOS 首版`。
> 优先级：P0 数据/主路径，P1 高频能力，P2 可后补能力。
> 旧实现路径现仅部分保留于 `legacy/backend/src/...` 与 `legacy/frontend/src/...`；未保留模块见 `legacy/README.md`。

## 1. 剪贴板采集

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| C-01 | P0 | 系统剪贴板监听 | `clipboard/watcher.rs` | 后台轮询 `NSPasteboard.changeCount`，120ms | 已通过 | `TestWatcherReportsChangesOnly` + 真机连续复制 |
| C-02 | P0 | 纯文本采集 | `read.rs`、`ingest.rs` | `NSPasteboard.string(forType:)` | 已通过 | 真机复制中英文、多行 |
| C-03 | P0 | HTML 采集 | `read.rs`、`ingest.rs` | `public.html` 读取，plain 作文本回退 | 已通过 | 真机 `NSPasteboardItem` 写入 html + plain |
| C-04 | P0 | RTF 采集 | `read.rs`、`ingest.rs` | `public.rtf` 读取，plain 作文本回退 | 已通过 | `TestCaptureOrderAndTypeSwitches`；待真机 Pages/Word |
| C-05 | P0 | 图片采集 | `read.rs`、`storage.rs` | PNG 直通，其他 `NSImage` 转 PNG | 已通过 | 真机 NSImage → TIFF → PNG，尺寸 520×520 |
| C-06 | P0 | 文件/文件夹采集 | `read.rs`、`ingest.rs` | `public.file-url` 列表 | 已通过 | 真机多文件 URL；`TestCaptureFiles` |
| C-07 | P0 | 采集优先级 | `settings::Capture.order` | 同一 pasteboard 多表示时按配置选主类型 | 已通过 | `TestCaptureOrderAndTypeSwitches` |
| C-08 | P0 | 类型开关 | `settings::Capture` | 五类内容独立开关 | 已通过 | 同上，关闭 RTF 后回退 HTML |
| C-09 | P0 | 文本大小上限 | `max_text_mb` | 字节数判断，0 为不限 | 已通过 | `TestTextSizeLimit` |
| C-10 | P0 | 图片大小上限 | `max_image_mb` | 解码前/后大小判断，0 为不限 | 已通过 | `TestImageSizeLimit` |
| C-11 | P0 | 内容去重 | `db/items::upsert_item` | `blake3("<kind>:<content>")` 命中更新 | 已通过 | 真机重复复制 `use_count` 2；数据库测试 |
| C-12 | P0 | 来源应用同步抓取 | `clipboard/source.rs` | 监听回调内取 frontmost app | 已通过 | 真机来源记录为 `com.openai.codex` |
| C-13 | P1 | 来源应用图标 | `clipboard/icon.rs`、`app_store.rs` | `NSRunningApplication` 取图标，128px PNG 去重缓存 | 已通过 | `TestAppIconStoreIsIdempotent`、`TestDownscalePNG`、`TestAppIconStoreRemovesUnreferenced`；真机 Finder 图标缓存为 9.4KB / 128×128 |
| C-14 | P0 | 忽略应用 | `Filters.excluded_app_ids` | bundle id 命中即跳过 | 已通过 | 默认排除密码类 App；`TestIgnoredAppIsNotCaptured` |
| C-15 | P0 | 敏感内容识别 | `clipboard/secrets.rs` | 私钥、token、AWS key、JWT、标签 secret | 已通过 | `TestContainsSecret` 固定样本矩阵 |
| C-16 | P0 | 敏感内容策略 | `collectSecrets`、`redactSecrets` | 跳过或脱敏，日志不落原文 | 已通过 | `TestSecretCapturePolicy`、`TestPreviewRedactsSensitiveText` |
| C-17 | P0 | 回环抑制 | `clipboard/guard.rs` | 写回前登记所有实际写出的表示 hash，2 秒 TTL | 已通过 | `TestWriteItemSuppressesItsOwnCapture`、`TestWriteRichTextSuppressesHTMLAndPlainFallback`、`TestWriteFilesAsTextSuppressesFallbackCapture`；真机复制 HTML 后 DB 不新增纯文本回环 |
| C-18 | P1 | 监听暂停 | `WatcherPause`、托盘 | 暂停期间跳过入库，watcher 不退出 | 已通过 | 真机托盘暂停后复制不入库，恢复后继续监听；`TestWatcherPollBypassesPause` |
| C-19 | P2 | 复制音效 | `clipboard/sound.rs` | `AVAudioPlayer` 或 Go 音频库 | 已通过（产品调整） | `TestCopySoundIsAvailable`、`TestSettingsModelAppliesPaths`；真机开启“复制提示音”后系统 `Pop` 音效可用，关闭后不再播放；按产品调整改用系统音效，见迁移计划 |
| C-20 | P1 | 手动读取剪贴板 | `read_clipboard` | 与自动采集共用同一条管线 | 已通过 | `TestWatcherPollReadsOnlyChanges`；真机暂停时读取入库，恢复后不重复；冷启动前已有剪贴板内容可由托盘读取入库 |

## 2. 数据、历史与存储

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| D-01 | P0 | SQLite 主库 | `db/init.rs`、`db/path.rs` | `modernc.org/sqlite`，同路径同 schema | 已通过 | `go test ./internal/database`：新库创建、旧时间格式读取 |
| D-02 | P0 | Migration 兼容 | `_sqlx_migrations`、`0001_init.sql` | 识别旧 migration，不重复建表 | 已通过 | 新库与仅含 `clipboard_items` 的旧库都会创建/补齐 `_sqlx_migrations` v1，保留已有 SQLx 元数据；以 SQLx CLI 对新建库执行 `migrate run` 无重复迁移；`TestOpenRecordsLegacyMigrationBaseline`、`TestOpenAddsBaselineToDatabaseWithoutMigrationTable`、`TestOpenKeepsExistingMigrationMetadata` |
| D-03 | P0 | 全文搜索 | FTS5 trigram | 同 FTS 表与触发器 | 已通过 | `TestSearchUsesFTSAndLikeFallback` |
| D-04 | P0 | 短词搜索 | `LIKE` 兜底 | 1–2 字符走 `LIKE` | 已通过 | 中文单字、英文双字测试 |
| D-05 | P0 | 历史筛选 | `db/items.rs` | kind/group/favorite/pinned/keyword | 已通过 | 仓储层 `Query` 已覆盖；原生分类/自定义分组筛选真机通过 |
| D-06 | P0 | 排序 | `settings.clipboard.content.sort` | 置顶优先，再按时间/使用次数 | 已通过 | `TestPagePaginationAndPinnedFirst` |
| D-07 | P0 | 分页 | `ClipboardItemPage` | `list + total + has_more` | 已通过 | `Page` 返回 total/hasMore |
| D-08 | P0 | 收藏 | `toggle_item_favorite` | `is_favorite`，不刷新 `updated_at` | 已通过 | `TestMetadataUpdatesKeepUpdatedAt` |
| D-09 | P1 | 自动收藏 | `autoFavorite` | 写非空备注时联动收藏 | 已通过 | `TestEditNoteNormalizesAndAutoFavorites`；清空备注不触发 |
| D-10 | P0 | 置顶 | `toggle_item_pinned` | `is_pinned`，排序最前 | 已通过 | 排序测试覆盖 |
| D-11 | P0 | 备注 | `update_item_note` | 更新 note，不刷新 `updated_at` | 已通过 | `SetNote` + `updated_at` 不变测试 |
| D-12 | P0 | 分组 | `db/groups.rs` | 原生创建、重命名、筛选；保留 schema 的图标/隐藏字段 | 已通过（产品调整） | `TestCreateGroupAppendsAndListsInOrder`、`TestRenameGroupCancelAndDuplicate`；真机创建“工作”、重命名为“项目”成功 |
| D-13 | P0 | 分组布局 | `update_clipboard_groups_layout` | 首版按创建顺序稳定展示，不做拖拽排序/隐藏 | 不纳入 macOS 首版（产品调整） | 决定与理由见 `03-migration-plan.md` Phase 4 分组产品调整 |
| D-14 | P0 | 删除分组 | `delete_group` | 条目 `group_id` 置 NULL | 已通过 | `TestDeleteGroupKeepsItemsAndClearsGroupID`；真机删除后 15 条记录保留，SQLite 校验 `group_id = NULL` |
| D-15 | P0 | 使用次数 | `increment_item_use_count` | `use_count + 1` 并刷新 `updated_at` | 已通过 | `TestUpsertDeduplicatesAndRefreshesUpdatedAt` |
| D-16 | P0 | 历史清理 | `cleanup.rs`、`db/items.rs` | 时长 + 最大条数，收藏/置顶保留 | 已通过 | `TestCleanupHistoryByRetentionKeepsProtectedItems`、`TestCleanupRemovesExpiredImage` |
| D-17 | P0 | 启动清理 | `clipboard::init` | 启动先跑一次，再按间隔 | 已通过 | `TestCleanupDue` 覆盖禁用、1 小时边界、过期与溢出；`TestCleanupLoopRunsOnConfiguredInterval` 用可注入时钟验证启动先跑一次且按配置间隔再次执行；隔离数据目录插入普通/收藏/近期三条，冷启动后仅删除过期普通项，收藏与近期项保留 |
| D-18 | P0 | 清空记录 | `clear_items` | 可选保留收藏/置顶 | 已通过 | 托盘原生对话框接取消、保留收藏/置顶、清除全部；`TestClearHistoryOptionallyKeepsFavoritesAndPinned` 校验行级策略，`TestClearRemovesOnlyDeletedImageFiles` 校验保留项图片不被删除、清除全部时对应图片同步移除 |
| D-19 | P0 | 数据根解析 | `core/paths.rs` | `<app_local_data>/<dev\|prod>` + `storage.json` | 已通过 | `internal/storage` 测试覆盖默认根、锚点与自定义目录 |
| D-20 | P0 | 自定义数据目录 | `commands/storage.rs` | 迁移 db/resources/config/state 并热切换 | 已通过 | `TestChangeDataDirCopiesLiveData`、`TestChangeDataDirRemovesTargetOnFailure`、`TestRestoreDefaultDataDir`；真机自定义目录迁移、恢复默认目录（生成 `dev-backup-*`）与已存在目标错误路径提示均通过，失败不重启 |
| D-21 | P0 | 资源目录 | `core/paths.rs` | 同目录结构，图片按 hash 分片 | 已通过 | `TestImageStoreIsIdempotent`；真机图片落盘 |
| D-22 | P1 | 存储占用 | `get_storage_usage` | total/database/resources/settings 分项 | 已通过 | `TestMeasureUsageSumsParts`；真机托盘显示 136.9 KB，与 `du` 结果可解释 |
| D-23 | P1 | 缓存清理 | `clean_resource_cache` | 删除未被 DB 引用的图片/图标 | 已通过 | `TestCleanCacheKeepsReferencedFiles` 覆盖图片、应用图标、文件图标；真机清理孤儿文件且保留引用 |
| D-24 | P1 | 旧版数据导入 | `commands/onboarding.rs` | 读 v0.6 `history` + `images/` | 不纳入 macOS 首版（产品调整） | 首版不提供 v0.6 数据目录导入，`internal/onboarding` 已移除；旧版用户先导出 `.ecopastebak`，再通过新版备份导入恢复数据，`TestImportBackupsWrittenByLegacyApp` 覆盖旧版明文/加密备份兼容 |
| D-25 | P1 | 来源应用注册表 | `db/apps.rs`、`apps_registry.rs` | 列表、全部应用、从路径添加、清理未引用 | 已通过 | `TestListAppsOrdersByNameAndRefreshesMetadata`、`TestDeleteUnreferencedAppsKeepsReferencedRows`、`TestAppRegistryMergesRunningApps`、`TestAppRegistryAddRefreshesWithoutDuplicates`；真机应用页勾选立即写入 `excludedAppIds`，添加 Notes 自动排除并落库，取消勾选删除未引用行与图标 |
| D-26 | P1 | 文件图标缓存 | `db/file_icons.rs`、`file_icon_store.rs` | `NSWorkspace` 图标 + cache key | 已通过 | `TestFileIconCacheReusesType`；真机 PDF/TXT 图标正确，两个 PDF 共享 `.pdf` 缓存 |
| D-27 | P1 | 备份入口 | `commands/backup.rs` | 导出、检查、导入、pending slot | 已通过 | 托盘明文/加密导出、merge/overwrite 导入与 pending slot；设置“数据”页复用同一导出/导入流程，真机验证导出选择、保存面板与导入文件面板均可进入并取消；`TestSettingsModelRunsBackupActions` 覆盖动作分派；`TestMergeImport` 校验跳过重复、补齐缺失资源、不覆盖已有资源和设置 patch，`TestStageAndApplyImport`、`TestApplyImportRestoresOnFailure` 校验覆盖与回滚；真机 merge 导入计数、DB/resources/设置落盘，overwrite 导入后重启校验；旧版 Rust 算法生成的明文/加密样本由 `TestImportBackupsWrittenByLegacyApp` 导入，Go 导出包由同一 Rust 读取逻辑交叉校验 |
| D-28 | P1 | 打开数据目录 | `commands/storage.rs` | `NSWorkspace` 打开目录 | 已通过 | 真机从托盘打开当前数据根 |

## 3. 内容识别与预览

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| P-01 | P1 | URL 识别 | `clipboard/detect.rs` | 带协议或 `www.`、单行 | 已通过 | `TestDetectTextSubKind`；真机复制 `https://example.com/open-test` 入库 `sub_kind=url` |
| P-02 | P1 | Email 识别 | `detect.rs` | 常见 local-part 字符，支持中文 | 已通过 | `TestDetectTextSubKind` 覆盖 ASCII/中文/标签邮箱与反例 |
| P-03 | P1 | 颜色识别 | `detect.rs` | hex/rgb/hsl/hwb/lab/lch/oklab/oklch/color-mix/gradient | 已通过 | `TestDetectTextSubKind` 覆盖 hex、rgb、gradient 与注入反例 |
| P-04 | P1 | 路径识别 | `detect.rs` | 仅绝对路径且存在 | 已通过 | `TestDetectTextSubKind`；真机复制存在的目录后入库 `sub_kind=path` |
| P-05 | P1 | 列表摘要 | `ingest.rs` | trim 后最多 256 字符 | 已通过 | `summarize` + `TestPageStripsTextContent` |
| P-06 | P1 | 列表内容裁剪 | `commands/clipboard.rs` | 列表不返回完整 HTML/RTF content | 已通过 | `Page` 对 text 行返回空 content |
| P-07 | P1 | 图片缩略图 | `storage.rs` | 最长边 300px，缺失时懒生成 | 已通过 | `TestImageStoreCreatesBoundedThumbnail`、`TestEnsureThumbnailBackfillsMissingFile`；真机 3456×2234 截图使用 300×193 缩略图 |
| P-08 | P1 | 文件条目 | `commands/clipboard.rs` | 路径、名称、目录/图片标记、存在性、大小 | 已通过 | `TestDescribeFileEntries`；真机列表显示 PDF/TXT/图片大小与“已丢失” |
| P-09 | P1 | 文件预览模式 | `FilesPreviewKind` | 单图片走图片预览，其余列表 | 已通过 | 单图片、单非图片、多文件测试覆盖；真机图片文件 Space 预览通过 |
| P-10 | P1 | 预览窗口 | `window/preview.rs`、`pages/Preview` | 原生 `NSPopover` 锚定所选行，Space/Esc，文本/图片/文件 | 已通过 | 真机文本/HTML/图片预览锚定所选行弹出，面板保持可见；Space 与 Esc 关闭并回到搜索框，点击外部按原生临时 popover 行为关闭；预览态回车沿用粘贴路径并触发既有辅助功能提示，`⌘C` 复制所选条目；行内文本不拦截点击，点击正文可直接选中并预览该行；预览浮层使用原生背景，与面板半透明材质分离；`TestPreviewSizeFollowsTextHeight`、`TestPreviewSizeClampsLongText`；产品调整记录在迁移计划 |
| P-11 | P2 | HTML/RTF 富文本预览 | 当前只用纯文本摘要 | `NSAttributedString` + `NSTextView`（增强项） | 已通过 | 真机 Space 预览 HTML/RTF，粗体、字号、颜色和段落生效，Esc 返回；HTML 列表标记不再与 AppKit 文本列表重复绘制，显式彩色文本保留原色，未指定或默认黑白色使用语义 `labelColor` 适配深浅色；敏感条目继续使用脱敏纯文本且不传递原始富文本；`TestPreviewIncludesRichText`、`TestAttributedPreview`、`TestAttributedPreviewDropsImportedListMarkers`、`TestAttributedPreviewPreservesExplicitColor`、`TestPreviewInkNeedsLabelForDefaultContrast`；仅渲染文本样式，不加载外部资源 |

## 4. 历史复用动作

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| A-01 | P0 | 写回纯文本 | `clipboard/write.rs` | `NSPasteboard` plain text | 已通过 | 真机选中列表项 Enter 写入，`pbpaste` 校验一致 |
| A-02 | P0 | 写回 HTML | `write.rs` | plain + `public.html` 同一 item | 已通过 | 真机复制 `<b>rich hello</b>` 后剪贴板含 HTML + plain，`pbpaste` 得到 `rich hello`，DB 不新增回环 |
| A-03 | P0 | 写回 RTF | `write.rs` | plain + `public.rtf` 同一 item | 已通过 | 隔离实例复制历史 RTF 条目后，`NSPasteboard` 同时提供 `public.utf8-plain-text` 与 `public.rtf`，RTF 数据 148 B；未新增自身回环 |
| A-04 | P0 | 写回图片 | `write.rs` | `public.png` + TIFF（按需） | 已通过 | 隔离实例复制 520×520 历史图片后，剪贴板提供 `public.png` 5467 B，并由系统合成 `public.tiff`；未新增自身回环 |
| A-05 | P0 | 写回文件 | `write.rs` | `public.file-url` 多文件 | 已通过 | 隔离实例复制两项文件记录后，`NSPasteboard` 按 item 提供 `file:///tmp/ecopaste-a.txt` 与 `file:///tmp/ecopaste-b.txt`，同时提供 `NSFilenamesPboardType` |
| A-06 | P0 | 粘贴 | `paste_clipboard_item` | 写回 + 让出 key + `CGEvent` ⌘V | 已通过 | 真机 TextEdit 空文稿中选中历史文本后按回车，目标收到完整原文，面板按时隐藏，数据库无自身回环；`TestHistoryKeyAction` 覆盖回车/Enter 进入粘贴路径；拒绝权限提示路径沿用已验流程 |
| A-07 | P0 | 纯文本粘贴 | `paste_plain` | 文本只写 plain；HTML/RTF 带 plain 回退 | 已通过 | 真机对含粗体和红色的 HTML 条目按 `⌥⌘V`，剪贴板仅提供 `public.utf8-plain-text`/`NSStringPboardType`，TextEdit 收到纯文本；固定面板下粘贴后保持可见并恢复表格焦点；菜单“纯文本粘贴/复制为纯文本”此前已验证 |
| A-08 | P1 | 文件路径粘贴 | `paste_files_as_path` | 文件记录写回换行路径文本 | 已通过 | `TestBuildWritePreservesRepresentations`；真机“粘贴为路径”与 `pasteFilesAsPath=true` 正常粘贴均写入换行路径，未新增回环 |
| A-09 | P1 | 打开链接/邮件 | `open_clipboard_item_link` | `NSWorkspace.open` | 已通过 | `TestOpenDispatchesBySubKind`；URL/路径条目右键菜单显示对应动作 |
| A-10 | P1 | Finder 显示 | `reveal_clipboard_item` | `NSWorkspace.activateFileViewerSelecting` | 已通过 | `TestRevealFilesAndPath`；真机对路径条目执行后 Finder 选中目标目录 |
| A-11 | P1 | 图片另存 | `save_clipboard_image_to_file` | `NSSavePanel`，默认文件名规则 | 已通过 | `TestSaveImageWritesChosenFile`；真机保存 3456×2234 PNG，文件头与尺寸校验通过 |
| A-12 | P1 | 拖出文件 | `drag_out/macos.rs` | `NSTableView` 拖拽源 + file URL，多文件在 `willBegin` 一次性写入全部 URL | 已通过 | `TestDragOffersFiles`、`TestWriteDragFilesWritesEveryURL`；真机将双文件条目拖入 Finder 目标目录，两个落盘文件均与源文件 SHA-256 一致 |
| A-13 | P1 | 拖出图片 | `drag_out/macos.rs` | 拖拽源写 `public.png` | 已通过 | `TestDragOffersImageBytes`；真机从面板拖出 128×128 PNG 到 TextEdit，文稿生成原生 `AXAttachment`，图片内容保留 |
| A-14 | P1 | 拖出文本 | `drag_out/macos.rs` | plain/HTML/RTF 多表示同一 `NSPasteboardItem` | 已通过 | `TestDragOffersRichText`；真机将 HTML 条目拖入 TextEdit，明文与标记一致；辅助功能读取到粗体 Times Bold 与红色前景，证明富文本表示未被纯文本回退替换 |
| A-15 | P1 | 备注编辑 | `NoteModal` | 原生备注弹窗，256 字符上限 | 已通过 | `TestEditNoteNormalizesAndAutoFavorites`、`TestEditNoteCancelAndClear`、`TestEditNoteLimitsLength`；保存/取消/空值/自动收藏已覆盖 |
| A-16 | P1 | 移动到分组 | 右键子菜单 | `NSMenu` 动态分组列表与当前分组勾选 | 已通过 | `TestMoveToGroupAndBackToUngrouped`；真机移动到“工作”后分组筛选仅显示 1 条 |
| A-17 | P0 | 删除条目 | `delete_clipboard_item` | 删除 DB 行 + 图片文件 | 已通过 | 右键/Delete 按设置确认；`TestDeleteRemovesImageFile`、`TestDeletePolicyHonorsProtectionSettings`；收藏/置顶删除入口按 `deleteFavoriteItems`/`deletePinnedItems`/`deleteFavoriteItemsOnlyInFavoriteGroup` 隐藏，普通/收藏/置顶分别使用独立确认开关 |
| A-18 | P1 | 自动粘贴 | `Content.auto_paste` | 单击/双击、粘贴/复制、禁用 | 已通过 | `TestClickActionModes`；真机 `singleClickCopy` 单击写入、双击不触发，`doubleClickCopy` 单击不触发、双击写入 |
| A-19 | P1 | 中键动作 | `Content.middle_click` | 粘贴/纯文本粘贴/复制/纯文本复制/禁用 | 已通过 | `TestMiddleClickActions`；真机合成中键点击列表行，在清空剪贴板后触发纯文本复制并写入目标内容；粘贴模式复用 A-06 辅助功能路径 |
| A-20 | P1 | 悬停快捷动作 | `Content.item_actions` | 可配置、可排序 | 已通过 | `TestHoverActionsFollowConfiguredOrder`、`TestHoverActionsSkipInapplicableRows`、`TestHoverActionReflectsItemState`、`TestQuickActionsKeepConfiguredOrder`、`TestSettingsModelAppliesItemActions`；真机悬停行显示配置按钮并即时刷新状态；设置“动作”页勾选“编辑备注”后 `itemActions` 立即落盘，上移后 `itemActionOrder` 更新且启用的动作按新顺序排列；动作条使用 `NSPopover` 材质圆角浮层，图标尺寸与点击区域同步放大，选中行上仍保持清晰边界 |
| A-21 | P1 | 复制后隐藏 | `copyThenHideWindow` | 复制后按设置隐藏，固定窗口时保持可见 | 已通过 | `TestHideAfterActionRespectsPinnedPanel`；真机固定后复制面板保持，取消固定后复制面板隐藏 |

## 5. 窗口与系统集成

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| W-01 | P0 | 剪贴板面板 | `window/macos.rs` | 双 `NSPanel`：顶部分类栏子窗口 + 主面板，均非激活、置顶、透明磨砂 | 迁移中 | 顶部分类栏为独立 `NSPanel`，通过 `addChildWindow:ordered:` 挂在主面板上方，保持 10pt 间距、同宽并随主窗口移动/显示/隐藏；两个窗口各自使用 `NSVisualEffectView` 磨砂与独立圆角，深色外观下面板为通透中灰而非发黑；分类胶囊、筛选菜单、设置与固定按钮、卡片式历史列表和失焦隐藏已真机验证；多屏定位、负坐标屏幕和“可见区域小于面板”边界由 `TestPanelOrigin*` 覆盖；当前环境仅单屏，多屏切换仍未真机复核；主面板与顶部分类栏的空白区域支持按住拖动整组窗口，移动在超过 3pt 阈值后开始，阈值内仍按点击处理 |
| W-02 | P0 | 窗口定位 | `window/position.rs` | 跟随鼠标/居中/记住位置 | 已通过 | 设置“面板打开位置”三档；真机 `osascript` 读取窗口 frame：居中 `654,217`、跟随光标 `300,393`（光标 `300,400`）、记住位置恢复 `654,217` |
| W-03 | P0 | 失焦隐藏 | `window/macos.rs` | 自定义 resign-key 隐藏 + modal/固定窗口期间暂停 | 已通过 | 真机切换 Finder 后隐藏；系统另存/备注弹窗期间保持；固定窗口保持可见；拖拽期间暂停失焦隐藏，拖放被接受后隐藏，取消拖拽保持；`TestShouldHideAfterDrag` 覆盖接受、取消、失 key、失活、固定和抑制分支 |
| W-04 | P0 | 固定窗口 | `set_clipboard_window_pinned` | 固定不隐藏；粘贴时 resign/restore key | 已通过 | 原生按钮与 `⌘P` 切换已接，`⌘T` 置顶选中条目；真机固定后失焦、切换 TextEdit 和回车粘贴均保持可见，粘贴时让出 key 并在完成后恢复表格焦点；`TestHistoryKeyAction` 覆盖两档快捷键 |
| W-05 | P1 | 窗口生命周期 | `window/lifecycle/*` | 原生窗口状态机，无 WebView 重建 | 已通过 | 冷启动不显示面板、全局快捷键显示/隐藏、隐藏后写 `window-state.json` 已实测；快捷键切换改为显式 `Show()`/`Hide()`，避免 Carbon 回调中的窗口命令未刷新；Esc 隐藏已复测；修复退出阶段热键清理阻塞主队列导致进程残留后，应用菜单退出、托盘“重新启动”与托盘“退出 EcoPaste”均真机通过：重启产生新进程，退出在 0.78 秒内结束且无残留 |
| W-06 | P1 | 窗口几何持久化 | `window/state.rs` | 同 `state/window-state.json` | 已通过（格式调整） | 隐藏时写入 `clipboard` 帧并重启恢复；旧版存物理像素、左上原点，新版存 AppKit points、左下原点，JSON 字段与路径不变；旧坐标不在任一屏幕时回退到光标所在屏幕居中 |
| W-07 | P1 | 偏好窗口 | `window/mod.rs` | 原生侧栏 + `NSTabView` 设置窗口 | 已通过 | 左侧原生源码列表分类栏、右侧内容区；已打开窗口切换分类时同步选中侧栏行；顶部 `NSSearchField` 跨全部分类搜索标题、说明、选项、路径，结果下拉跳转并在目标行短暂高亮；`TestMatchSettingsSearch*` 覆盖多词、路径分词、去重和 8 条上限，真机验证 `dock`、`capture order` 跳转及无匹配状态 |
| W-08 | P1 | 更新窗口 | `window/mod.rs` | Sparkle 原生更新窗口 | 迁移中 | 产品调整：不恢复独立 WebView 更新窗，入口合并到设置“更新”页、应用菜单和托盘；macOS 由 Sparkle 显示检查/下载/安装/重启界面，应用侧只传本地化不可用结果；真机开发版验证设置页与菜单入口，正式签名 appcast 升级待发布验证 |
| W-09 | P1 | 引导窗口 | `window/mod.rs` | 不提供首启引导浮层 | 不纳入 macOS 首版（产品调整） | 冷启动不再自动展开面板或显示引导；权限缺失在粘贴动作和设置“权限”页提示；引导代码与 `onboarding` 设置字段已移除，见 S-14 |
| W-10 | P0 | 托盘 | `tray/mod.rs` | `NSStatusItem` + `NSMenu`：偏好、暂停、开源、更新、版本、重启、退出 | 迁移中 | 显示/暂停/设置/清除/开源/更新/版本/重启/退出已接；更新项与应用菜单共用同一 Sparkle 检查路径，正式签名 appcast 升级待发布验证 |
| W-11 | P0 | 全局快捷键 | `shortcut/mod.rs` | `desktop.RegisterHotKey` | 已通过 | Carbon 注册支持热更新；只增量替换变化的按键，按快捷键返回注册失败并在设置页显示，失败时恢复原值；`TestSettingsModelRollsBackShortcutWhenRebindFails`；真机录制 `Alt+J` 后立即重绑，未影响另一快捷键 |
| W-12 | P1 | 快捷键录制 | 前端录制器 | 原生 `NSTextField` 捕获 | 已通过 | 原生输入框捕获组合键，Delete 清空、Esc 取消；`TestSettingsModelAppliesShortcutAndRejectsDuplicate`；真机快捷键页完成录制与写盘 |
| W-13 | P1 | 开机自启 | `autostart/macos.rs` | `SMAppService` | 迁移中 | `TestAutostartNeedsChange`；真机开关、持久化通过；重启登录需签名安装包验收 |
| W-14 | P1 | Dock 图标显隐 | `show_taskbar_icon` | `NSApplication.setActivationPolicy` | 已通过 | `TestSettingsModelAppliesDockIcon`；真机切换立即生效，重启后恢复持久化状态 |
| W-15 | P1 | 单实例 | `tauri-plugin-single-instance` | 数据根 `flock` + `OnReopen` | 已通过 | `TestLockInstanceRejectsSecond`；真机二次进程 0.1s 退出，主实例打开设置；`open` 重入同样打开设置 |
| W-16 | P1 | 文件关联 | `tauri.conf.json`、`backup/*` | `nsgo.yaml file_types` + `OnOpenFiles` | 已通过 | `com.ayangweb.eco-paste.backup` 写入 Info.plist；真机运行中与冷启动 `open -a` 均触发导入，导入后重启校验 DB |
| W-17 | P1 | 原生右键菜单 | `menu/clipboard_item.rs` | `NSMenu` 按内容动态生成动作 | 已通过 | URL/图片/路径菜单与分组子菜单已真机验证；`⌘C`/`⌘V`/`⌥⌘C`/`⌥⌘V` 与标准编辑菜单复制/粘贴已真机验证 |
| W-18 | P1 | 权限 | `PermissionControl` | 辅助功能/完全磁盘访问检测与跳转 | 已通过 | `TestFullDiskAccessGranted`；权限页真机显示两项实时状态，FDA 跳转到“完全磁盘访问权限”面板，粘贴失败仍弹辅助功能说明 |
| W-19 | P2 | 防默认行为 | `core/prevent_default.rs` | 无 WebView，不需要 | 不纳入 macOS 首版 | 确认无 WebView 入口 |
| W-20 | P1 | 更新 | `update.rs` | NSGo updater + Sparkle appcast | 迁移中 | 已接 `core.CheckForUpdates` 与 NSGo updater：macOS 由 Sparkle 负责检查、EdDSA 校验、下载、安装和重启；应用菜单、托盘和设置“更新”页共用入口，非 Sparkle 结果使用本地化提示；`TestDescribeUpdateResult`、`TestSettingsModelChecksUpdates`、`TestApplicationUpdateItem`；真机开发版验证入口与不可用提示，`nsgo.yaml` 的 `updates.key` 与测试 appcast/真实升级仍待发布密钥配置 |
| W-21 | P1 | 日志 | `tauri-plugin-log` | 标准 `log` + lumberjack 轮转，写入前脱敏 | 已通过 | `TestRedact`、`TestOpenWritesRedactedLogFile`；真机落盘 `<data_root>/logs/ecopaste.log`，托盘可打开日志目录 |

## 6. UI 页面与交互

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| U-01 | P0 | Clipboard 主页面 | `legacy/frontend/src/pages/Clipboard` | AppKit 双窗口：分类胶囊栏 + 搜索/列表/卡片/Footer | 已通过（产品调整） | 分类移入独立顶部子窗口，展示全部/文本/图片/文件/收藏胶囊与新建按钮；主窗口保留搜索、筛选菜单、设置、固定按钮、圆角卡片历史列表与 Footer；类型标签、来源/时间、置顶/收藏状态、备注切换、悬停动作、搜索/分组筛选、实时刷新与复制已真机验证；行卡片默认显示轻量底色，选中态使用系统强调色并切换白色文字；真机 420×560 主面板 + 420×48 分类栏复核；`TestHistoryRowIncludesMetadata`、`TestHistoryRowTypeLabels`、`TestPanelSearchWidthLeavesRoomForToolbar`、`TestPanelNavCapsulesFitPanelWidth` |
| U-02 | P1 | Preview 页面 | `legacy/frontend/src/pages/Preview` | 原生 `NSPopover` 预览 | 已通过 | 真机 Space 打开、Space/Esc 关闭，popover 锚定列表行且不遮挡面板；文本、HTML、图片预览均正常；`TestPreviewRedactsSensitiveText` |
| U-03 | P1 | Preference 页面 | `legacy/frontend/src/pages/Preference` | 原生侧栏/表单/搜索/高亮 | 已通过 | 左侧 `NSVisualEffectView` + 源码列表分类栏承载 14 个分类，右侧原生表单、全局搜索和高亮已可用；分类行支持键盘聚焦、选中强调和 VoiceOver“设置分类”标签，搜索和语言切换会同步选中侧栏行；真机验证全部分类切换、`backup` 跳转高亮及中英文重建，侧栏显示打包后的 EcoPaste 应用图标；“采集”页按内容类型、大小限制和优先级分组，中英文“上移/下移”按钮与表格宽度完整可用；“显示”页按列表密度分组，并提供文本/图片双栏原生行预览，随文本行数与图片高度实时更新；“交互”“删除”“窗口”“历史与隐私”页按操作、权限和内容类型分组，并补充自动换行的原生说明文案；“窗口”页补齐“打开时聚焦搜索”和“下次打开时清空搜索”，与现有打开行为共同覆盖 `clipboard.search` 和 `clipboard.window` 的 schema 设置；“动作”“应用”页加入分区标题和自动换行说明，上移/下移按钮的 VoiceOver 文案按页面区分；动作、应用与采集优先级列表统一使用语义色圆角内嵌容器，动作与应用列表改用 AppKit 原生 checkbox，真机验证勾选状态即时落盘并恢复，VoiceOver 标签保持；采集优先级和动作顺序按钮会按首行/末行禁用；开关行标题宽度与截断处理已修正，避免英文长标题被裁切；窗口分类图标修正为可读的 `rectangle.on.rectangle`；“数据”页按存储位置与备份恢复分组，数据目录单独占行完整展示并提供 tooltip；“权限”页按系统权限与诊断信息分组，中英文按钮宽度已适配；“预览”页控制 Space 预览，“更新”页展示版本并触发 Sparkle，说明文案改用原生多行标签并在英文长度下完整换行；`lightweightMode`/`idleDestroySeconds`、hover 预览、Sparkle 更新字段和 Windows 专用项按产品调整不暴露，其余 schema 可配置项均已覆盖 |
| U-04 | P1 | Onboarding 页面 | `legacy/frontend/src/pages/Onboarding` | 不提供独立引导页面 | 不纳入 macOS 首版（产品调整） | 产品调整：移除首启引导流程与相关面板浮层，`internal/onboarding`、对应 UI、i18n 与设置字段均已删除；权限、快捷键、排除应用与备份恢复分别由设置页、系统权限提示和备份导入承担 |
| U-05 | P1 | Update 页面 | `legacy/frontend/src/pages/Update` | 原生更新状态/进度 | 迁移中 | 产品调整：不再自绘更新状态机，设置“更新”页展示当前版本与检查入口，下载进度、安装和重启交给 Sparkle 原生窗口；不可用或非 Sparkle 结果使用本地化提示；正式签名 appcast 升级待发布验证 |
| U-06 | P2 | ContextMenu 页面 | `legacy/frontend/src/pages/ContextMenu` | macOS 使用 `NSMenu`，不迁移 WebView 菜单窗 | 参考 | 对比菜单结构 |
| U-07 | P0 | 列表虚拟滚动 | `react-virtuoso` | `NSTableView` 原生虚拟化 | 已通过 | `NSTableView` 原生复用与 80 条分页；隔离数据目录导入 1 万条记录，首屏 80 条、原生滚动加载至 160 条、连续 End 导航加载至 240 条，搜索 `9999` 约 1.3 秒返回 1 条，无崩溃 |
| U-08 | P1 | 键盘导航 | `legacy/frontend/src/hooks` 的键盘逻辑 | 上下/Page/Home/End/Enter/Esc，以及打开/收藏/备注/新建分组/粘贴前 10 项 | 已通过 | 真机 ↑/↓、Home/End、⌘F、⌘C、⌥⌘C、⌘V、⌥⌘V 通过；向表格发送原生 PageDown/PageUp 键码 121/116，选择从无选中跳到第 5 条、再回到第 1 条，页大小按可见行计算；兼容专用键与 Fn+↑/↓，并支持 macOS 首/末行快捷键 Fn+←/→ 与 ⌘↑/↓；`⌘D` 收藏、`⌘M` 编辑备注、`⌘N` 新建分组已真机验证，焦点在搜索框时同样生效，`⌘M` 由面板 event monitor 抢先处理以免被窗口菜单的最小化截获；`⌘O` 按内容打开链接/邮件或在 Finder 显示路径与文件，`⌘Enter` 纯文本粘贴，`⌘1–0` 粘贴前 10 项，分别由 `TestHistoryKeyAction`、`TestCommandEnterPastesPlain`、`TestCommandDigitIndex` 覆盖；真机按 `⌘1` 进入既有辅助功能权限提示，确认按键已路由 |
| U-09 | P1 | 主题 | antd token + `light/dark` | AppKit 语义色 + `NSAppearance` | 已通过 | `TestSettingsModelAppliesTheme`；真机设置内切换深色立即生效并持久化，重启后面板与设置窗口保持深色；浅色下次启动验证 |
| U-10 | P1 | 动画 | `motion` | AppKit 动画/`Core Animation` | 已通过（产品调整） | 面板打开使用 0.14s 淡入；预览改用系统 `NSPopover` 自带过渡，减少重复动画；开启“减少动态效果”时面板直接显示；`TestAnimateInKeepsViewVisibleAndAddsFade`，真机面板与 HTML/RTF 预览正常显示、Space/Esc 返回 |
| U-11 | P2 | 滚动条 | `overlayscrollbars` | 系统 `NSScrollView` 滚动条 | 不纳入 macOS 首版（产品调整） | 原生 `NSScrollView` 已按系统“显示滚动条”偏好自动选择 overlay/legacy，强制 overlay 会覆盖用户系统设置；旧版自定义滚动条不再迁移 |
| U-12 | P2 | 诊断 | `pages/Preference` 诊断区 | 原生诊断面板/复制信息 | 已通过 | “权限”页展示版本、macOS 版本、数据目录与两项权限状态，“复制诊断信息”写入剪贴板并显示反馈；真机 `pbpaste` 校验五项内容完整；`TestFormatDiagnosticReport` |
| U-13 | P2 | 快捷键速查 | `Clipboard/components/ShortcutList.tsx` | Footer 键盘按钮 + `NSPopover` 快捷键列表，⌘K 打开 | 已通过 | Footer 新增 `keyboard` 图标按钮，⌘K 与点击均可打开原生 popover；真机弹出“快捷键”列表，Esc 关闭，按钮位于 Footer 右侧且不挤压记录数；列表覆盖粘贴、纯文本粘贴、前 10 项、预览、复制、打开、收藏、置顶、备注、删除、导航、搜索、固定窗口、新建分组与关闭；文案走 i18n 目录，`TestShortcutHelpItemsAreComplete` 校验条目完整与按键不重复 |

## 7. 国际化与文案

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| I-01 | P1 | 前端文案 | `legacy/frontend/src/locales/zh-CN`、`en-US` | Go i18n 目录，复用文案内容 | 已通过 | 剪贴板面板与 14 个设置分类已接入目录：搜索/分类/分组/空状态、右键与悬停动作、备注与分组弹窗、预览提示、类型/时间摘要、文件摘要、全部设置项/选项/搜索关键词/诊断报告、更新入口与结果提示；备份/清除/权限弹窗，以及历史、分组、数据目录、快捷键、来源应用、链接打开、登录项与备份容器错误均已接入；`TestHistoryPresentationFollowsLanguage`、`TestSettingsSearchMatchesEnglishLabels`、`TestFormatDiagnosticReportInEnglish`、`TestGroupValidationFollowsLanguage`、`TestBackupErrorsFollowLanguage` 覆盖英文 |
| I-02 | P1 | Rust 侧文案 | `src-tauri/src/i18n/` | Go 文案目录 | 已通过 | `internal/i18n` 中英文目录覆盖托盘、列表右键、清除/删除确认、备份/存储/缓存/数据目录提示、剪贴板面板、设置与权限弹窗，并按 `appearance.language` 选择；`TestCatalogsShareEveryKey`、`TestSetLanguageSelectsCatalog`、`TestTfFormatsArguments` |
| I-03 | P1 | 语言切换 | `appearance.language` | 原生菜单/设置切换，立即生效 | 已通过 | 设置“通用”页语言下拉写入 `appearance.language` 后立即切换目录、重建设置窗口与剪贴板面板，无需重启；真机中英文往返验证，托盘菜单、面板、设置与清空/备份弹窗文案同步切换；`TestSettingsModelAppliesLanguage` |
| I-04 | P2 | 系统本地化 | 部分系统菜单 | AppKit 本地化角色项 | 已通过 | `internal/menu` 提供“应用/编辑/窗口”原生菜单与角色项，语言切换时整体重建；`TestApplicationBuildsLocalizedMenuBar`、`TestApplicationUsesRoleLabels`、`TestApplicationSettingsItem`；真机中英文往返验证关于/设置/隐藏/退出、撤销/重做/剪切/复制/粘贴/全选、最小化/缩放/全屏/关闭均随系统语言更新，设置项可从菜单打开偏好窗口 |

## 8. 设置项

设置模型以 `legacy/backend/src/settings/model.rs` 为准，落盘 `config/settings.json`，全部 `serde(default)`、camelCase。

| ID | 设置组 | 字段 | 迁移要求 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- |
| S-01 | general | `autoStart`、`runAsAdmin`、`trayIcon`、`dockIcon` | `autoStart`/`trayIcon`/`dockIcon` 已接系统副作用；`runAsAdmin` 仅 Windows 参考 | 已通过（产品调整） | `TestSettingsModelAppliesTrayIcon`；真机开发版切换“显示菜单栏图标”后状态即时落盘，重启后保持隐藏，再次开启可恢复图标；默认值、原子写回与重置副作用沿用设置层测试 |
| S-02 | appearance | `theme`、`language` | `auto/light/dark`、`zh-CN/en-US` | 已通过 | 主题三档已接入并即时应用；`TestSettingsModelAppliesTheme`；语言下拉即时重建界面并持久化，真机中英文往返回读 `settings.json` 正确，`TestSettingsModelAppliesLanguage` |
| S-03 | shortcuts | `openClipboard`、`openPreference`、`winV` | 保留；`winV` 仅 Windows 参考 | 已通过 | 两个快捷键均已读取并注册；“快捷键”页原生录制、清空、非法组合和重复校验可用；`TestSettingsModelAppliesShortcutAndRejectsDuplicate`、`TestSettingsModelRollsBackShortcutWhenRebindFails`；真机 `Alt+K` 改为 `Alt+J` 后立即写盘并重绑 |
| S-04 | clipboard.capture | `text/html/rtf/image/files`、`maxTextMb`、`maxImageMb`、`order` | 采集顺序与上限一致 | 已通过 | `TestCaptureOrderAndTypeSwitches`、`TestSettingsModelAppliesCaptureOrder`；真机设置“采集”页选择纯文本上移/下移后，`settings.json` 即时写入完整五类顺序 |
| S-05 | clipboard.content | `autoPaste`、`middleClick`、`copyPlain`、`copyThenHideWindow`、`pastePlain`、`pasteFilesAsPath`、`showOriginalPreview`、删除保护、`autoFavorite`、`updateOnReuse`、`sort`、`itemActions`、`itemActionOrder` | 行为语义逐项一致 | 已通过 | 纯文本复制/粘贴、文件路径粘贴、复制后隐藏、复用排序、排序方式、中键动作、单击/双击触发已接入；设置“交互”页可改左键/中键动作、文件路径粘贴、备注自动收藏和“悬停备注条目显示原始内容”，真机写盘验证通过；有备注条目默认显示备注、真实鼠标移入后切回原内容并在移出后恢复备注，`TestHistoryRowTextUsesNoteUntilHover` 覆盖同一逻辑；“删除”页接入普通/收藏/置顶确认与删除保护，列表、右键、悬停和 Delete 键共用同一策略；悬停动作支持勾选与上下排序 |
| S-06 | clipboard.display | `textMaxLines`、`imageMaxHeight`、`fileMaxCount` | 列表展示上限一致：文本标题行数、图片缩略图高度、文件名数量；因 purego 不支持 CGFloat 回调，列表使用统一行高`max(58, 文本行高, 图片高度+24)` | 已通过 | `TestDisplayClampsStoredDensity`、`TestHistoryItemHonorsFileMaxCount`；设置“显示”页可调；真机 `textMaxLines=3` 时 89pt 行高完整显示多行长文本与元数据，`imageMaxHeight=64` 时图片缩略图完整、无裁切，切换文本/图片分类后列表密度一致 |
| S-07 | clipboard.sensitive | `collectSecrets`、`redactSecrets` | 两种策略互斥语义一致 | 已通过 | 设置 UI 可切换；敏感样本与预览脱敏测试 |
| S-08 | clipboard.history | `retention{value,unit}`、`maxCount`、`cleanupIntervalHours` | 清理规则一致 | 已通过 | 设置 UI 可调；`TestCleanupHistoryByRetentionKeepsProtectedItems`、`TestCleanupHistoryByMaxCount` |
| S-09 | clipboard.search | `defaultFocus`、`clearOnHide` | 面板打开/隐藏行为一致 | 已通过 | 设置“窗口”页提供“打开时聚焦搜索”和“下次打开时清空搜索”，写入 `clipboard.search.*` 后由面板观察器立即生效；真机验证表格/搜索框初始焦点、失焦隐藏后保留查询、隐藏后清空查询并恢复完整列表；`TestSettingsModelAppliesPaths` 覆盖路径映射与回读 |
| S-10 | clipboard.window | `position`、`scrollToTopOnOpen`、`selectRangeOnOpen`、`selectCategoryOnOpen`、`selectGroupOnOpen`、`lightweightMode`、`idleDestroySeconds` | 原生窗口等价行为；无 WebView 时轻量化/空闲销毁按“不适用”处理，字段继续读写 | 已通过 | `position` 三档已真机验证；`TestOpenBehavior*`、`TestOpenSelectionAppliesCategoryAndGroup` 覆盖滚动/范围/分类/分组及组合优先级；真机设置“分类=图片 + 分组=全部”冷启动后图片分类选中且仅显示 1 条图片，设置“分组=测试打开”冷启动后分组下拉选中该组且仅显示组内 1 条，分类与分组选择不再互相清空 |
| S-11 | clipboard.preview | `hoverEnabled`、`hoverDelayMs`、`spaceEnabled` | Space 按开关生效；hover 按迁移计划不提供 | 已通过（产品调整） | `TestSpacePreviewHonorsSetting`；设置“预览”页关闭后 Space 不打开预览，开启后立即写入 `clipboard.preview.spaceEnabled` 并恢复 Space 预览；设置页改用原生浮层的操作说明，不再暴露迁移计划文案；真机确认 `NSPopover` 按空间出现在面板左侧或右侧，Esc/Space 返回并保持当前行选中；短 HTML 预览按内容收缩至 406×158，长内容滚动到底完整可见，切换条目后回到顶部；默认字体为系统无衬线字体，保留粗体、显式颜色与列表，深色模式下默认深色墨水自动切换语义正文色；`TestAttributedPreviewKeepsNativeDefaultFont`、`TestAttributedPreviewPreservesExplicitColor`、`TestPreviewInkNeedsLabelForDefaultContrast`、`TestPreviewSizeFollowsTextHeight` 覆盖；悬停预览不实现的原因见 `03-migration-plan.md` Phase 4 |
| S-12 | clipboard.feedback | `copySound` | 音效开关一致 | 已通过 | 设置“通用”页开关即时写入 `settings.json`；开启后监听/手动读取入库成功播放系统提示音，关闭后静音；`TestSettingsModelAppliesPaths` 覆盖读写 |
| S-13 | clipboard.filters | `excludedAppIds` | bundle id 过滤一致 | 已通过 | `TestDefaultExcludedAppsAreNotCaptured` 校验默认排除密码类应用且普通来源照常入库；`TestIgnoredAppIsNotCaptured`、`TestSettingsModelAppliesExcludedAppsAndDeletesRemoved`、`TestSettingsModelAddSourceAppExcludesIt` 覆盖自定义排除、取消与来源应用清理 |
| S-14 | onboarding | `completed`、`lastStep`、`legacyImport{checked,imported,importTypes,importedAt}` | 移除引导/导入设置 | 不纳入 macOS 首版（产品调整） | 产品调整：开发阶段不承担旧版兼容，`Onboarding` 结构与 `settings.json` 的 `onboarding` 字段已移除；未知 JSON 字段仍由解码器忽略，不影响其它设置读写 |
| S-15 | update | `autoCheck`、`frequency`、`includeBeta`、`includeNightly`、`lastCheckedAt`、`skippedVersion` | Sparkle 管理自动检查与跳过；旧字段保持读写 | 迁移中（产品调整） | macOS 首版只接单一 stable appcast，自动检查、下载与跳过版本由 Sparkle 管理；beta/nightly 需要独立 appcast 与发布渠道，暂不暴露；旧 `update` 字段继续按发布契约读写，`TestSettingsJSONRoundTripsPublishedCamelCase` 保证不丢字段；测试 appcast 与真实签名升级待发布验证 |
| S-16 | 默认值 | 所有 `serde(default)` | 缺字段时行为与旧版一致 | 已通过 | `TestLoadKeepsDefaultsForMissingFields`、`TestOpenFallsBackToDefaultsForUnreadableSettings`、`TestSettingsJSONRoundTripsPublishedCamelCase`：缺字段、空文件、损坏 JSON、类型错误和非法枚举均回退完整默认值，camelCase 编码可无损回环 |
| S-17 | 写入语义 | 对象深合并、数组替换、`.tmp` + rename | patch 语义与原子写一致 | 已通过 | `TestUpdateDeepMergesObjectsAndReplacesArrays`、`TestWriteIsAtomicAndReadable`、`TestConcurrentUpdatesKeepCompleteSettings`、`TestCrashResidueDoesNotAffectSettings`；并发 patch 不丢字段，残留 `.tmp` 不影响主文件读取，下次写入完成替换且不残留临时文件 |
| S-18 | 跨字段校验 | 快捷键不重复、`selectGroupOnOpen` 编码合法 | 错误可定位且不破坏旧值 | 已通过 | `TestUpdateRejectsInvalidValuesWithoutChangingSettings`、`TestUpdateRejectsDuplicateShortcuts`：非法枚举、快捷键冲突和非法打开分组均拒绝，内存快照与磁盘文件保持原值 |
| S-19 | 重置 | `reset_settings` | 恢复默认但保留必要身份/数据；设置窗口底部确认后立即生效 | 已通过 | `TestResetSettingsRestoresDefaultsAndSideEffects`：默认值、Dock、自启、主题与快捷键/列表刷新均恢复；数据目录与历史不受影响 |

## 9. 数据契约与兼容

| ID | 优先级 | 契约 | 现有定义 | 迁移要求 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| X-01 | P0 | SQLite schema | `migrations/0001_init.sql` | 表、列、索引、触发器一致 | 已通过 | `internal/database/schema.sql` 直接复制已发布 migration |
| X-02 | P0 | `updated_at` 语义 | 最近一次复用时间 | 备注/收藏/置顶/分组不刷新 | 已通过 | `TestMetadataUpdatesKeepUpdatedAt` |
| X-03 | P0 | `content_hash` | `blake3("<kind>:<content>")` | 字节级一致 | 已通过 | `TestHashMatchesPublishedFormat` |
| X-04 | P0 | 图片文件名 | `<blake3>.png` 分片 | 路径与命名一致 | 已通过 | `TestImageStoreUsesPublishedBLAKE3NameAndShard`：BLAKE3 空输入向量、文件名 `<digest>.png`、`origin/<digest[:2]>/` 路径与旧存储布局一致 |
| X-05 | P0 | `settings.json` | camelCase、serde default | 字段、默认值、patch 语义一致 | 已通过 | `TestSettingsJSONRoundTripsPublishedCamelCase`、`TestLoadKeepsDefaultsForMissingFields`、`TestOpenFallsBackToDefaultsForUnreadableSettings`、`TestWriteIsAtomicAndReadable`、`TestCrashResidueDoesNotAffectSettings`：发布字段 camelCase 无损回环，旧文件缺字段保留默认值，更新后重新打开可读且结构完整 |
| X-06 | P0 | 数据目录 | `storage.json` 锚点 | 默认/自定义目录均可解析 | 已通过 | `TestResolveReadsPublishedCustomManifest` 直接读取旧版 camelCase `storage.json` 与 `.ecopaste-storage.json` 并解析自定义根；`TestSetRootPointsManifestAtCustomDir`、`TestResolveFallsBackFromBrokenManifest`、`TestResolveRejectsForeignIdentity` 覆盖重写、损坏回退和环境隔离 |
| X-07 | P0 | `.ecopastebak` | ZIP / 自定义加密容器 | 双向读写兼容 | 已通过 | `internal/backup/testdata` 固定旧版 Rust 算法生成的明文/加密样本，`TestImportBackupsWrittenByLegacyApp` 校验 ZIP 路径、manifest、数据库、设置、资源和 Argon2id/XChaCha20-Poly1305；Go 导出的两种容器均由同一 Rust 读取逻辑交叉验证；merge/overwrite 导入与 rollback 继续由 `internal/backup` 测试覆盖 |
| X-08 | P0 | 文件关联 | `.ecopastebak` | 同扩展名/UTI | 已通过 | `nsgo.yaml` 声明 `com.ayangweb.eco-paste.backup`，构建产物 Info.plist 校验 `CFBundleDocumentTypes`；真机运行中/冷启动 `open -a` 均进入导入确认，导入后重启校验 DB |
| X-09 | P0 | Bundle ID | `com.ayangweb.eco-paste` | 正式版一致；开发版隔离 | 迁移中 | `make dev`/`make build-dev` 使用 `com.ayangweb.eco-paste.dev`；开发构建误用正式 ID 时启动退出；开发 bundle ID 在 `ECOPASTE_MODE` 之前映射到 `dev` 数据根，避免覆写正式数据；`lsof` 实测开发实例只访问 `dev/db`、`dev/logs`、`dev/state`；正式版安装/升级与登录项迁移待签名包验收 |

## 10. Windows/Linux 专用能力

当前不进入 macOS 首版实现，但必须在退役前明确处置，避免“以为迁移完了其实删掉了后续平台基础”。

| ID | 功能 | 现有实现 | 当前处置 | 后续条件 |
| --- | --- | --- | --- | --- |
| N-01 | Windows 剪贴板监听 | `clipboard/watcher.rs` Windows 分支 | 保留参考 | 启动 Windows 适配时重写 |
| N-02 | Windows 低级键盘钩子 | `keyboard/windows.rs` | 保留参考 | Windows 主窗口键盘导航 |
| N-03 | Windows 全局鼠标钩子 | `mouse/windows.rs` | 保留参考 | 失焦隐藏 |
| N-04 | Windows 窗口实现 | `window/windows.rs` | 保留参考 | 窗口/菜单窗 |
| N-05 | Windows 拖出 | `drag_out/windows*.rs` | 保留参考 | 拖放 |
| N-06 | Windows 粘贴注入 | `keystroke/windows.rs` | 保留参考 | `SendInput` |
| N-07 | Win+V 替代 | `shortcut/win_v.rs` | 保留参考 | Windows 快捷键 |
| N-08 | Windows 自启 | `autostart/windows.rs` | 保留参考 | 注册表/提权 |
| N-09 | Windows 管理员启动 | `admin.rs` | 保留参考 | UAC |
| N-10 | Windows 菜单窗 | `menu/context_window.rs`、`pages/ContextMenu` | 保留参考 | 避免菜单抢焦点 |
| N-11 | Windows 构建/发布 | `.github/workflows/*` Windows job | 保留 | 平台适配阶段重写 |
| N-12 | Linux 支持 | 当前无实现 | 不新增 | 未来单独立项 |

## 11. 验收记录规则

- 每完成一项，把状态改为 `已通过`，并在验证列补充实际命令、截图路径或手动步骤编号。
- 任何“与旧版不同”的行为必须写明差异和理由，不能只写“已通过”。
- 新增功能先加入本表，再实现，避免 checklist 与代码脱节。
- 退役 Tauri 前，P0/P1 不允许存在 `待迁移` 或 `迁移中`；P2 例外必须记录。
