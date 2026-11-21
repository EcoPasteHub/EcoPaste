# 04 · 功能 Parity 清单

> 用途：迁移期间逐项勾对，防止“代码搬完了但功能少了一块”。
> 状态说明：`待迁移`、`迁移中`、`已通过`、`参考`（只保留作为旧实现参考）、`不纳入 macOS 首版`。
> 优先级：P0 数据/主路径，P1 高频能力，P2 可后补能力。
> 旧实现路径现仅部分保留于 `legacy/backend/src/...` 与 `legacy/frontend/src/...`；未保留模块见 `legacy/README.md`。

> 2.0.0 是兼容性基线；旧版仅供功能参考，不作为数据或行为的兼容要求。

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
| C-18 | P1 | 监听暂停 | `WatcherPause`、托盘 | 暂停停止采样 ticker，以事件恢复，watcher 等待事件而不退出 | 已通过 | 既有真机托盘暂停/恢复验证；`TestWatcherPollBypassesPause`、`TestWatcherStopsPollingWhilePausedAndResumes`、`TestCanceledPollDoesNotReadPasteboard` 覆盖无轮询、恢复、手动读取及取消 |
| C-19 | P2 | 复制音效 | `clipboard/sound.rs` | `AVAudioPlayer` 或 Go 音频库 | 已通过（产品调整） | `TestCopySoundIsAvailable`、`TestSettingsModelAppliesPaths`；真机开启“复制提示音”后系统 `Pop` 音效可用，关闭后不再播放；按产品调整改用系统音效，见迁移计划 |
| C-20 | P1 | 手动读取剪贴板 | `read_clipboard` | 与自动采集共用同一条管线 | 已通过 | `TestWatcherPollReadsOnlyChanges`；真机暂停时读取入库，恢复后不重复；冷启动前已有剪贴板内容可由托盘读取入库 |

## 2. 数据、历史与存储

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| D-01 | P0 | SQLite 主库 | `db/init.rs`、`db/path.rs` | `modernc.org/sqlite`；空库初始化，时间统一写为定长 UTC | 已通过 | `go test ./internal/database` 覆盖新库创建与仓储操作 |
| D-02 | P0 | Schema 版本 | 历史 SQLx 仅供参考 | SQLite `user_version=1`；拒绝未标记的现存 schema | 已通过 | `TestImportRejectsUnsupportedBackupBeforeChangingLocalData` 验证旧快照拒绝 |
| D-03 | P0 | 全文搜索 | FTS5 trigram | 同 FTS 表与触发器 | 已通过 | `TestSearchUsesFTSAndLikeFallback` |
| D-04 | P0 | 短词搜索 | `LIKE` 兜底 | 1–2 字符走 `LIKE` | 已通过 | 中文单字、英文双字测试 |
| D-05 | P0 | 历史筛选 | `db/items.rs` | kind/group/favorite/pinned/keyword | 已通过 | 仓储层 `Query` 已覆盖；原生分类/自定义分组筛选真机通过 |
| D-06 | P0 | 排序 | `settings.clipboard.content.sort` | 置顶优先，再按时间/使用次数 | 已通过 | `TestPagePaginationAndPinnedFirst` |
| D-07 | P0 | 分页 | `ClipboardItemPage` | `list + total + has_more` | 已通过 | `Page` 返回 total/hasMore |
| D-08 | P0 | 收藏 | `toggle_item_favorite` | `is_favorite`，不刷新 `updated_at` | 已通过 | `TestMetadataUpdatesKeepUpdatedAt` |
| D-09 | P1 | 自动收藏 | `autoFavorite` | 写非空备注时联动收藏 | 已通过 | `TestSaveNoteNormalizesAndAutoFavorites`；清空备注不触发 |
| D-10 | P0 | 置顶 | `toggle_item_pinned` | `is_pinned`，排序最前 | 已通过 | 排序测试覆盖 |
| D-11 | P0 | 备注 | `update_item_note` | 更新 note，不刷新 `updated_at` | 已通过 | `SetNote` + `updated_at` 不变测试 |
| D-12 | P0 | 分组 | `db/groups.rs` | 原生创建、重命名、筛选；只保存当前需要的分组字段 | 已通过（产品调整） | `TestCreateGroupNormalizesAndSelects`、`TestCreateGroupRejectsEmptyLongAndDuplicateNames`、`TestRenameGroupCancelAndDuplicate`；新建分组改为锚定分类栏「+」按钮的 `NSPopover` 内联输入框（自动聚焦、回车提交、Esc/点击外部取消），不再使用居中模态对话框；重命名/删除仍为以剪贴板面板为基准居中并通过模态 run loop 抬到面板之上的对话框，`TestCenterDialogOrigin*` 覆盖居中和屏幕边界 |
| D-13 | P0 | 分组布局 | `update_clipboard_groups_layout` | 首版按创建顺序稳定展示，不做拖拽排序/隐藏 | 不纳入 macOS 首版（产品调整） | 决定与理由见 `03-migration-plan.md` Phase 4 分组产品调整 |
| D-14 | P0 | 删除分组 | `delete_group` | 条目 `group_id` 置 NULL | 已通过 | `TestDeleteGroupKeepsItemsAndClearsGroupID`；真机删除后 15 条记录保留，SQLite 校验 `group_id = NULL` |
| D-15 | P0 | 使用次数 | `increment_item_use_count` | `use_count + 1` 并刷新 `updated_at` | 已通过 | `TestUpsertDeduplicatesAndRefreshesUpdatedAt` |
| D-16 | P0 | 历史清理 | `cleanup.rs`、`db/items.rs` | 时长 + 最大条数，收藏/置顶保留 | 已通过 | `TestCleanupHistoryByRetentionKeepsProtectedItems`、`TestCleanupRemovesExpiredImage` |
| D-17 | P0 | 启动与定时清理 | 内置调度 | 仅启用保留时长或数量限制时清理 | 已通过（自动化） | 启用时启动清理一次，之后每 15 分钟运行；规则变化立即清理，全部禁用时没有清理调用和定时器；设置观察器仅阻塞等待规则变化；`TestCleanupLoopRunsOnBuiltInInterval`、`TestCleanupLoopFollowsEnabledLimits`、`TestUnrelatedSettingsDoNotTriggerCleanup` 覆盖生命周期与退出 |
| D-18 | P0 | 清空记录 | `clear_items` | 可选保留收藏/置顶 | 已通过 | 面板顶部垃圾桶与托盘共用原生对话框，接取消、保留收藏/置顶、清除全部；`TestClearHistoryOptionallyKeepsFavoritesAndPinned` 校验行级策略，`TestClearRemovesOnlyDeletedImageFiles` 校验保留项图片不被删除、清除全部时对应图片同步移除；真机面板点击垃圾桶弹出确认框，取消后记录不变 |
| D-19 | P0 | 数据根解析 | `core/paths.rs` | `<app_local_data>/<dev\|prod>` + `storage.json` | 已通过 | `internal/storage` 测试覆盖默认根、锚点与自定义目录 |
| D-20 | P0 | 自定义数据目录 | `commands/storage.rs` | 迁移 db/resources/config/state 并热切换 | 已通过 | `TestChangeDataDirCopiesLiveData`、`TestChangeDataDirRemovesTargetOnFailure`、`TestRestoreDefaultDataDir`；真机自定义目录迁移、恢复默认目录（生成 `dev-backup-*`）与已存在目标错误路径提示均通过，失败不重启 |
| D-21 | P0 | 资源目录 | `core/paths.rs` | 同目录结构，图片按 hash 分片 | 已通过 | `TestImageStoreIsIdempotent`；真机图片落盘 |
| D-22 | P1 | 存储占用 | `get_storage_usage` | total/database/resources/settings 分项 | 已通过 | `TestMeasureUsageSumsParts`；真机托盘显示 136.9 KB，与 `du` 结果可解释 |
| D-23 | P1 | 缓存清理 | `clean_resource_cache` | 删除未被 DB 引用的图片/图标 | 已通过 | `TestCleanCacheKeepsReferencedFiles` 覆盖图片、应用图标、文件图标；真机清理孤儿文件且保留引用 |
| D-24 | P1 | 旧版数据导入 | `commands/onboarding.rs` | 2.0.0 不迁移旧目录或导入旧备份 | 不纳入 | 用户自行清理；当前版本备份见 D-27 |
| D-25 | P1 | 来源应用注册表 | `db/apps.rs`、`apps_registry.rs` | 列表、全部应用、从路径添加、清理未引用 | 已通过 | `TestListAppsOrdersByNameAndRefreshesMetadata`、`TestDeleteUnreferencedAppsKeepsReferencedRows`、`TestAppRegistryMergesRunningApps`、`TestAppRegistryAddRefreshesWithoutDuplicates`；真机应用页勾选立即写入 `excludedAppIds`，添加 Notes 自动排除并落库，取消勾选删除未引用行与图标 |
| D-26 | P1 | 文件图标缓存 | `db/file_icons.rs`、`file_icon_store.rs` | `NSWorkspace` 图标 + cache key | 已通过 | `TestFileIconCacheReusesType`；真机 PDF/TXT 图标正确，两个 PDF 共享 `.pdf` 缓存 |
| D-27 | P1 | 备份入口 | `commands/backup.rs` | 导出、格式检查、合并、覆盖 | 已通过（自动化） | `TestStageAndApplyImport` 覆盖明文和加密覆盖恢复；`TestMergeImport` 覆盖去重、资源与设置；`TestImportRejectsUnsupportedBackupBeforeChangingLocalData` 验证拒绝旧格式和非当前 schema |
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
| P-09 | P1 | 文件预览模式 | `FilesPreviewKind` | 单图片按图片预览，无法读取时降级为文件行 | 已通过 | 单图片、单非图片、多文件测试覆盖；真机图片文件 Space 预览通过；单个图片文件仅在实际可解码时进入图片预览，未授予完全磁盘访问权限或文件被删除/损坏时降级为「图片图标 + 文件名」卡片，行高同步按文件行计算，不保留图片预览高度；`TestHistoryFileRowPreviewsReadableImage`、`TestHistoryFileRowFallsBackForUnreadableImage`、`TestFilePreviewUnreadableImageFallsBackToText`、`TestHistoryRowHeightUsesFileLayoutForUnreadableImage`、`TestFileKindSymbolMatchesFormat` |
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
| A-07 | P0 | 纯文本粘贴 | 已删除 | 粘贴保留原始文本格式；纯文本复制保留 | 已移除 | 设置、强制纯文本粘贴接口、菜单、悬停动作、快捷键分支和帮助行一并删除；复制偏好不再影响粘贴 |
| A-08 | P1 | 文件路径粘贴 | 已删除 | 文件记录始终写回原生文件对象 | 已移除 | 设置、菜单、快捷键、悬停动作及路径文本写回分支删除；`TestBuildWritePreservesRepresentations` 覆盖文件对象保留 |
| A-09 | P1 | 打开链接/邮件 | `open_clipboard_item_link` | `NSWorkspace.open` | 已通过 | `TestOpenDispatchesBySubKind`；URL/路径条目右键菜单显示对应动作 |
| A-10 | P1 | Finder 显示 | `reveal_clipboard_item` | `NSWorkspace.activateFileViewerSelecting` | 已通过 | `TestRevealFilesAndPath`；真机对路径条目执行后 Finder 选中目标目录 |
| A-11 | P1 | 图片另存 | `save_clipboard_image_to_file` | `NSSavePanel`，默认文件名规则 | 已通过 | `TestSaveImageWritesChosenFile`；真机保存 3456×2234 PNG，文件头与尺寸校验通过 |
| A-12 | P1 | 拖出文件 | `drag_out/macos.rs` | `NSTableView` 拖拽源 + file URL，多文件在 `willBegin` 一次性写入全部 URL | 已通过 | `TestDragOffersFiles`、`TestWriteDragFilesWritesEveryURL`；真机将双文件条目拖入 Finder 目标目录，两个落盘文件均与源文件 SHA-256 一致 |
| A-13 | P1 | 拖出图片 | `drag_out/macos.rs` | 拖拽源写 `public.png` | 已通过 | `TestDragOffersImageBytes`；真机从面板拖出 128×128 PNG 到 TextEdit，文稿生成原生 `AXAttachment`，图片内容保留 |
| A-14 | P1 | 拖出文本 | `drag_out/macos.rs` | plain/HTML/RTF 多表示同一 `NSPasteboardItem` | 已通过 | `TestDragOffersRichText`；真机将 HTML 条目拖入 TextEdit，明文与标记一致；辅助功能读取到粗体 Times Bold 与红色前景，证明富文本表示未被纯文本回退替换 |
| A-15 | P1 | 备注编辑 | `NoteModal` | 面板侧边 `NSPopover` 内联编辑，256 字符上限 | 已通过 | `TestSaveNoteNormalizesAndAutoFavorites`、`TestSaveNoteClears`、`TestSaveNoteLimitsLength`；保存/取消/空值/自动收藏已覆盖；备注改为多行 Popover：箭头与目标卡片垂直中心对齐，按屏幕空间显示在面板左侧或右侧，含灰色取消与蓝色确定按钮，`⌘M`、右键菜单与悬停动作共用 |
| A-16 | P1 | 移动到分组 | 右键子菜单 | `NSMenu` 动态分组列表与当前分组勾选 | 已通过 | `TestMoveToGroupAndBackToUngrouped`；真机移动到“工作”后分组筛选仅显示 1 条 |
| A-17 | P0 | 删除条目 | `delete_clipboard_item` | 删除 DB 行 + 图片文件 | 已通过 | 右键/Delete 按设置确认；`TestDeleteRemovesImageFile`、`TestDeletePolicyHonorsProtectionSettings`；收藏/置顶删除入口按 `deleteFavoriteItems`/`deletePinnedItems`/`deleteFavoriteItemsOnlyInFavoriteGroup` 隐藏，普通/收藏/置顶分别使用独立确认开关 |
| A-18 | P1 | 自动粘贴 | `Content.auto_paste` | 单击/双击、粘贴/复制、禁用 | 已通过 | `TestClickActionModes`；真机 `singleClickCopy` 单击写入、双击不触发，`doubleClickCopy` 单击不触发、双击写入 |
| A-19 | P1 | 中键动作 | 已删除 | 产品精简，不再提供中键快捷动作 | 已移除 | 中键事件、动作接口、设置、UI 与专属文案一并删除 |
| A-20 | P1 | 悬停快捷动作 | `Content.item_actions` | 可配置、可排序 | 已通过 | `TestHoverActionsFollowConfiguredOrder`、`TestHoverActionsSkipInapplicableRows`、`TestHoverActionReflectsItemState`、`TestQuickActionsKeepConfiguredOrder`、`TestSettingsModelAppliesItemActions`；真机悬停行显示配置按钮并即时刷新状态；设置“动作”页勾选“编辑备注”后 `itemActions` 立即落盘，上移后 `itemActionOrder` 更新且启用的动作按新顺序排列；动作条使用 `NSPopover` 材质圆角浮层，图标尺寸与点击区域同步放大，选中行上仍保持清晰边界 |
| A-21 | P1 | 复制后隐藏 | `copyThenHideWindow` | 复制后按设置隐藏，固定窗口时保持可见 | 已通过 | `TestHideAfterActionRespectsPinnedPanel`；真机固定后复制面板保持，取消固定后复制面板隐藏 |

## 5. 窗口与系统集成

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| W-01 | P0 | 剪贴板面板 | `window/macos.rs` | 双 `NSPanel`：顶部分类栏子窗口 + 主面板，均非激活、置顶、透明磨砂 | 迁移中 | 顶部分类栏为独立 `NSPanel`，通过 `addChildWindow:ordered:` 挂在主面板上方，保持 10pt 间距、同宽并随主窗口移动/显示/隐藏；两个窗口各自使用 `NSVisualEffectView` 磨砂与独立圆角，深色外观下面板为通透中灰而非发黑；分类胶囊、筛选菜单、设置、清除历史与固定按钮、卡片式历史列表和失焦隐藏已真机验证；多屏定位、负坐标屏幕、显示器间隙回退到最近屏幕、记忆位置部分越界钳制和“屏幕小于面板”边界由 `TestPanelOrigin*` 覆盖；定位只受屏幕 frame 约束，不再限制在可见工作区（可覆盖菜单栏/Dock 区域），由 `TestPanelOriginAllowsWorkAreaOverlap` 覆盖；当前环境仅单屏，多屏切换仍未真机复核；主面板与顶部分类栏的空白区域支持按住拖动整组窗口，移动在超过 3pt 阈值后开始，阈值内仍按点击处理 |
| W-02 | P0 | 窗口定位 | `window/position.rs` | 跟随鼠标/居中/记住位置 | 已通过 | 设置“面板打开位置”三档；真机 `osascript` 读取窗口 frame：居中 `654,217`、跟随光标 `300,393`（光标 `300,400`）、记住位置恢复 `654,217`；产品调整：定位仅受屏幕边界限制，取消工作区（`visibleFrame`）约束，面板可贴近或覆盖菜单栏/Dock 区域 |
| W-03 | P0 | 失焦隐藏 | `window/macos.rs` | 自定义 resign-key 隐藏 + modal/固定窗口期间暂停 | 已通过 | 真机切换 Finder 后隐藏；系统另存/备注 Popover 期间保持；固定窗口保持可见；拖拽期间暂停失焦隐藏，拖放被接受后隐藏，取消拖拽保持；`TestShouldHideAfterDrag` 覆盖接受、取消、失 key、失活、固定和抑制分支 |
| W-04 | P0 | 固定窗口 | `set_clipboard_window_pinned` | 固定不隐藏；粘贴时 resign/restore key | 已通过 | 原生按钮与 `⌘P` 切换已接，`⌘T` 置顶选中条目；真机固定后失焦、切换 TextEdit 和回车粘贴均保持可见，粘贴时让出 key 并在完成后恢复表格焦点；`TestHistoryKeyAction` 覆盖两档快捷键 |
| W-05 | P1 | 窗口生命周期 | `window/lifecycle/*` | 原生窗口状态机，无 WebView 重建 | 已通过 | 冷启动不显示面板、全局快捷键显示/隐藏、隐藏后写 窗口状态（见 W-06）；快捷键切换改为显式 `Show()`/`Hide()`，避免 Carbon 回调中的窗口命令未刷新；热键动作统一经 `Scope.Post` 投递到响应式图 goroutine，修复 Carbon 回调直接调用 `Show()` 触发的跨 goroutine panic（`TestHotKeyActionRunsOnGraphGoroutine`）；Esc 隐藏已复测；修复退出阶段热键清理阻塞主队列导致进程残留后，应用菜单退出、托盘“重新启动”与托盘“退出 EcoPaste”均真机通过：重启产生新进程，退出在 0.78 秒内结束且无残留 |
| W-06 | P1 | 窗口几何持久化 | `window/state.rs` | `state/panel-frame.json` 存单个 AppKit 面板 frame | 已通过（自动化） | `TestWindowStateRoundTrip`、`TestWindowStateDropsBrokenFile` |
| W-07 | P1 | 偏好窗口 | `window/mod.rs` | 原生侧栏 + `NSTabView` 设置窗口 | 已通过 | 左侧原生源码列表分类栏、右侧内容区；已打开窗口切换分类时同步选中侧栏行；已移除顶部设置项搜索入口，分类内的配置项保持不变 |
| W-08 | P1 | 更新窗口 | `window/mod.rs` | Sparkle 原生更新窗口 | 迁移中 | 产品调整：不恢复独立 WebView 更新窗，入口合并到设置“更新”页、应用菜单和托盘；macOS 由 Sparkle 显示检查/下载/安装/重启界面，应用侧只传本地化不可用结果；真机开发版验证设置页与菜单入口，正式签名 appcast 升级待发布验证 |
| W-09 | P1 | 引导窗口 | `window/mod.rs` | 不提供首启引导浮层 | 不纳入 macOS 首版（产品调整） | 冷启动不再自动展开面板或显示引导；权限缺失在粘贴动作和设置“权限”页提示；引导代码与 `onboarding` 设置字段已移除，见 S-14 |
| W-10 | P0 | 托盘 | `tray/mod.rs` | `NSStatusItem` + `NSMenu`：偏好、暂停、开源、更新、版本、重启、退出 | 迁移中 | 显示/暂停/设置/清除/开源/更新/版本/重启/退出已接；更新项与应用菜单共用同一 Sparkle 检查路径，正式签名 appcast 升级待发布验证 |
| W-11 | P0 | 全局快捷键 | `shortcut/mod.rs` | `desktop.RegisterHotKey` | 已通过 | Carbon 注册支持热更新；只增量替换变化的按键，按快捷键返回注册失败并在设置页显示，失败时恢复原值；`TestSettingsModelRollsBackShortcutWhenRebindFails`；真机录制 `Alt+J` 后立即重绑，未影响另一快捷键 |
| W-12 | P1 | 快捷键录制 | 前端录制器 | 原生 `NSTextField` 捕获 | 已通过 | 原生输入框捕获组合键，Delete 清空、Esc 取消；`TestSettingsModelAppliesShortcutAndRejectsDuplicate`；真机快捷键页完成录制与写盘 |
| W-13 | P1 | 开机自启 | `autostart/macos.rs` | `SMAppService` | 迁移中 | `TestAutostartNeedsChange`；真机开关、持久化通过；重启登录需签名安装包验收 |
| W-14 | P1 | Dock 图标显隐 | `show_taskbar_icon` | `NSApplication.setActivationPolicy` | 已通过（产品调整） | 移除设置项与 `general.dockIcon` 字段，改为偏好设置窗口打开时显示 Dock 图标、关闭最后一个设置窗口即隐藏；`internal/native/macos/settings.go` 的 `NewSettings`/`Close` 前后切换 activation policy；真机待验收 |
| W-15 | P1 | 单实例 | `tauri-plugin-single-instance` | 数据根 `flock` + `OnReopen` | 已通过 | `TestLockInstanceRejectsSecond`；真机二次进程 0.1s 退出，主实例打开设置；`open` 重入同样打开设置 |
| W-16 | P1 | 文件关联 | `tauri.conf.json`、`backup/*` | `nsgo.yaml file_types` + `OnOpenFiles` | 已通过 | `com.ayangweb.eco-paste.backup` 写入 Info.plist；真机运行中与冷启动 `open -a` 均触发导入，导入后重启校验 DB |
| W-17 | P1 | 原生右键菜单 | `menu/clipboard_item.rs` | `NSMenu` 按内容动态生成动作 | 已通过 | URL/图片/路径菜单与分组子菜单已真机验证；`⌘C`/`⌘V`/`⌥⌘C`/`⌥⌘V` 与标准编辑菜单复制/粘贴已真机验证 |
| W-18 | P1 | 权限 | `PermissionControl` | 辅助功能/完全磁盘访问检测与跳转 | 已通过 | `TestFullDiskAccessGranted`；权限页真机显示两项实时状态，FDA 跳转到“完全磁盘访问权限”面板，粘贴失败仍弹辅助功能说明 |
| W-19 | P2 | 防默认行为 | `core/prevent_default.rs` | 无 WebView，不需要 | 不纳入 macOS 首版 | 确认无 WebView 入口 |
| W-20 | P1 | 更新 | `update.rs` | NSGo updater + Sparkle appcast | 迁移中 | 已接 `core.CheckForUpdates` 与 NSGo updater：macOS 由 Sparkle 负责检查、EdDSA 校验、下载、安装和重启；应用菜单、托盘和设置“更新”页共用入口，非 Sparkle 结果使用本地化提示；`TestDescribeUpdateResult`、`TestSettingsModelChecksUpdates`、`TestApplicationUpdateItem`；真机开发版验证入口与不可用提示，`nsgo.yaml` 的 `updates.key` 与测试 appcast/真实升级仍待发布密钥配置 |
| W-21 | P1 | 日志 | `tauri-plugin-log` | 结构化记录、脱敏、5 MB 轮转压缩、可调级别及内部固定保留期 | 迁移中 | 日志系统内部固定保留 14 天；启动和每小时清理过期归档/崩溃日志，仍最多保留 20 个归档；活动主日志不按年龄删除；级别与打开目录入口保留。`TestOpenAppliesBuiltInLogRetention`、`TestLogCleanupKeepsNewestTwentyArchives`、原级别/脱敏/轮转/权限测试覆盖；真实崩溃验收未执行 |

## 6. UI 页面与交互

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| U-01 | P0 | Clipboard 主页面 | `legacy/frontend/src/pages/Clipboard` | AppKit 双窗口：分类胶囊栏 + 搜索/列表/卡片/Footer | 已通过（产品调整） | 分类移入独立顶部子窗口，展示全部/文本/图片/文件/收藏胶囊与新建按钮；主窗口保留搜索、筛选菜单、设置、固定按钮、圆角卡片历史列表与 Footer；类型标签、来源/时间、置顶/收藏状态、备注切换、卡片右上角绝对定位的悬停工具栏（纯色语义背景、统一过渡且不改变内容布局；行内状态更新保留工具栏，待真机回归）、搜索/分组筛选、实时刷新与复制已真机验证；行卡片默认显示轻量底色，选中态使用系统强调色并切换白色文字；真机 420×560 主面板 + 420×48 分类栏复核；卡片统一为左侧来源应用图标、右侧内容并移除图标角标的字体图标；文本卡片按配置行数自适应高度并在超出时省略，图片按最大高度等比缩放，非图片文件逐行显示文件图标与名称，单个图片文件按图片预览展示；Finder 搜索结果复制得到的 `/.file/id=` 文件引用在采集时解析为真实路径，不再误报“已丢失”；搜索栏上下与左右留白一致、内部元素垂直居中，移除自定义居中绘制后输入框聚焦前后位置不再偏移；顶部搜索栏区域改用单一 `panelHeaderMargin` 推导上下左右四个方向的留白：工具栏图标定义行高、搜索文字与图标共用同一垂直轴、列表首张卡片与图标之间保留同样间距，真机截图像素复核四个方向留白一致（原上下约 22/29pt、左右约 16/22pt）；列表滚动到底时最后一张卡片不再被裁掉：`NSTableView` 把每行放进「行高 + 行间距」的槽位并居中单元格，原文档高度按「行高之和 + (n-1) × 行间距」计算，比末行卡片底边少 1pt，滚动到底即被裁掉 1～2px，现改为补齐末行尾部行间距；`TestHistoryRowIncludesMetadata`、`TestHistoryRowTypeLabels`、`TestHistoryTableContentHeightFollowsRows`、`TestHistoryTableContentHeightCoversLastCard`、`TestPanelSearchWidthLeavesRoomForToolbar`、`TestPanelHeaderKeepsEqualMargins`、`TestPanelHeaderCentersSearchOnIconAxis`、`TestPanelNavCapsulesFitPanelWidth`；列表选中态始终保持有且仅有一项：切换分组/分类/搜索必定选中第一项（即使原选中项仍在当前列表），删除时选中下一项（无下一项则上一项），返回顶部（含打开时滚动到顶部）选中第一项；新建分组后停留在原分组，不自动切换；`TestSelectionIndexAlwaysKeepsOneRow` 覆盖；搜索结果在标题、预览与备注中命中关键词时以低饱和奶油黄高亮（不透明底、近黑文字，浅/深色自适应），未选中/选中外观均清晰可读，查询更新时列表原地重绘高亮；`TestHighlightRanges`、`TestHighlightTokensDedupesAndSplits`、`TestHighlightedRowStringAppliesBackgroundToMatches` 覆盖 |
| U-02 | P1 | Preview 页面 | `legacy/frontend/src/pages/Preview` | 原生 `NSPopover` 预览 | 已通过 | 真机 Space 打开、Space/Esc 关闭，popover 锚定列表行且不遮挡面板；文本、HTML、图片预览均正常；`TestPreviewRedactsSensitiveText` |
| U-03 | P1 | Preference 页面 | `legacy/frontend/src/pages/Preference` | 原生分类导航与表单 | 已通过 | 当前偏好设置按通用、外观、剪贴板、交互、数据、系统、关于七类组织。登录启动和快捷键归通用；主题、语言和列表密度归外观；采集类型/优先级与来源应用过滤归剪贴板；面板打开、搜索、复制粘贴与预览动作归交互；排序、保留策略、敏感内容、删除、存储与备份归数据；权限和日志诊断归系统；版本、更新和项目链接归关于。`TestEveryScalarSettingBelongsToOneCategory` 校验字段唯一归类，`TestSettingsPagesMatchNavigation` 校验导航页齐全；改组不改变设置 schema 路径。中英文标签通过共享 i18n catalog。
| U-04 | P1 | Onboarding 页面 | `legacy/frontend/src/pages/Onboarding` | 不提供独立引导页面 | 不纳入 macOS 首版（产品调整） | 产品调整：移除首启引导流程与相关面板浮层，`internal/onboarding`、对应 UI、i18n 与设置字段均已删除；权限、快捷键、排除应用与备份恢复分别由设置页、系统权限提示和备份导入承担 |
| U-05 | P1 | Update 页面 | `legacy/frontend/src/pages/Update` | 原生更新状态/进度 | 迁移中 | 产品调整：不再自绘更新状态机，设置“更新”页展示当前版本与检查入口，下载进度、安装和重启交给 Sparkle 原生窗口；不可用或非 Sparkle 结果使用本地化提示；正式签名 appcast 升级待发布验证 |
| U-06 | P2 | ContextMenu 页面 | `legacy/frontend/src/pages/ContextMenu` | macOS 使用 `NSMenu`，不迁移 WebView 菜单窗 | 参考 | 对比菜单结构 |
| U-07 | P0 | 列表虚拟滚动 | `react-virtuoso` | `NSTableView` 原生虚拟化 | 已通过 | `NSTableView` 原生复用与 80 条分页；隔离数据目录导入 1 万条记录，首屏 80 条、原生滚动加载至 160 条、连续 End 导航加载至 240 条，搜索 `9999` 约 1.3 秒返回 1 条，无崩溃；选择态刷新只遍历可视行（`rowsInRect:` 取可视范围），键盘连续导航不再随已加载条数线性变慢；加载下一页时复用已测量行高、仅测量新增页，追加时不再重测全部已加载项，并用 `insertRowsAtIndexes:withAnimation:` 增量插入而非整表 `reloadData`，既有 cell 与滚动位置保持不变、加载更多不再闪烁；切换分类/分组/搜索强制选中首行并滚动到顶部，避免沿用旧滚动偏移；`TestReusedRowHeightsOnlyExtendsUnchangedPrefix`、`TestRowsExtendRequiresUnchangedPrefix`、`TestCanAppendRows` 覆盖行高复用与追加判定边界 |
| U-08 | P1 | 键盘导航 | `legacy/frontend/src/hooks` 的键盘逻辑 | 上下/Page/Home/End/Enter/Esc，以及打开/收藏/备注/新建分组/粘贴前 10 项 | 已通过（产品精简） | 真机 ↑/↓、Home/End、⌘F、⌘C、⌥⌘C、⌘V 通过；向表格发送原生 PageDown/PageUp 键码 121/116，选择从无选中跳到第 5 条、再回到第 1 条，页大小按可见行计算；兼容专用键与 Fn+↑/↓，并支持 macOS 首/末行快捷键 Fn+←/→ 与 ⌘↑/↓；`⌘D` 收藏、`⌘M` 编辑备注、`⌘N` 新建分组已真机验证，焦点在搜索框时同样生效，`⌘M` 由面板 event monitor 抢先处理以免被窗口菜单的最小化截获；`⌘O` 按内容打开链接/邮件或在 Finder 显示路径与文件，粘贴统一保留原始文本格式；`⌘1–0` 粘贴前 10 项，分别由 `TestHistoryKeyAction`、`TestCommandDigitIndex` 覆盖；真机按 `⌘1` 进入既有辅助功能权限提示，确认按键已路由 |
| U-09 | P1 | 主题 | antd token + `light/dark` | AppKit 语义色 + `NSAppearance` | 已通过 | `TestSettingsModelAppliesTheme`；真机设置内切换深色立即生效并持久化，重启后面板与设置窗口保持深色；浅色下次启动验证 |
| U-10 | P1 | 动画 | `motion` | AppKit 动画/`Core Animation` | 已通过（产品调整） | 面板打开使用 0.14s 淡入；预览改用系统 `NSPopover` 自带过渡，减少重复动画；开启“减少动态效果”时面板直接显示；`TestAnimateInKeepsViewVisibleAndAddsFade`，真机面板与 HTML/RTF 预览正常显示、Space/Esc 返回 |
| U-11 | P2 | 滚动条 | `overlayscrollbars` | 系统 `NSScrollView` 滚动条 | 不纳入 macOS 首版（产品调整） | 原生 `NSScrollView` 已按系统“显示滚动条”偏好自动选择 overlay/legacy，强制 overlay 会覆盖用户系统设置；旧版自定义滚动条不再迁移 |
| U-12 | P2 | 诊断 | `pages/Preference` 诊断区 | 原生诊断面板/复制信息 | 已通过 | “权限”页展示版本、macOS 版本、数据目录与两项权限状态，“复制诊断信息”写入剪贴板并显示反馈；真机 `pbpaste` 校验五项内容完整；`TestFormatDiagnosticReport` |
| U-13 | P2 | 快捷键速查 | `Clipboard/components/ShortcutList.tsx` | Footer 键盘按钮 + `NSPopover` 快捷键列表，⌘K 打开 | 已通过（产品精简） | Footer 仅保留 `keyboard` 图标按钮，⌘K 与点击均可打开原生 popover；面板底栏不再显示常驻快捷键文字提示，完整列表由 popover 承载；列表覆盖粘贴、前 10 项、预览、复制、打开、收藏、置顶、备注、删除、导航、搜索、固定窗口、新建分组与关闭；纯文本粘贴条目随功能移除；`TestShortcutHelpItemsAreComplete` 校验条目完整与按键不重复 |

## 7. 国际化与文案

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| I-01 | P1 | 前端文案 | `legacy/frontend/src/locales/zh-CN`、`en-US` | Go i18n 目录，复用文案内容 | 已通过 | 剪贴板面板和偏好设置七个分类均由共享目录提供中英文文案；设置字段、选项、诊断报告、更新入口和结果提示，以及备份/清除/权限弹窗等文案均已接入；`TestHistoryPresentationFollowsLanguage`、`TestFormatDiagnosticReportInEnglish`、`TestGroupValidationFollowsLanguage`、`TestBackupErrorsFollowLanguage` 覆盖英文
| I-02 | P1 | Rust 侧文案 | `src-tauri/src/i18n/` | go-i18n 嵌套 JSON 文案目录 | 已通过 | `internal/i18n` 使用 `github.com/nicksnyder/go-i18n/v2` 加载内嵌 `locales/zh-CN.json`、`locales/en-US.json`，翻译键按功能模块组织为多层 JSON，由 go-i18n 展平为点分隔消息 ID；覆盖托盘、列表右键、清除/删除确认、备份/存储/缓存/数据目录提示、剪贴板面板、设置与权限弹窗，并按 `appearance.language` 选择；缺失文案回退 zh-CN；`TestCatalogFilesUseNestedObjects`、`TestCatalogsShareEveryKey`、`TestMissingTranslationFallsBackToDefault`、`TestSetLanguageSelectsCatalog`、`TestTfFormatsArguments` |
| I-03 | P1 | 语言切换 | `appearance.language` | 原生菜单/设置切换，原地响应式更新 | 已通过 | 设置“通用”页语言下拉写入 `appearance.language` 后，通过 NSGo Signal 更新语言状态；设置窗口标题、页签、表单标签/选项、面板工具栏/分类/列表行在原地更新，不关闭或重建窗口；真机中英文与跟随系统往返验证，窗口编号保持不变，托盘菜单、面板、设置与清空/备份弹窗文案同步切换；`TestSettingsModelAppliesLanguage` |
| I-04 | P2 | 系统本地化 | 部分系统菜单 | AppKit 本地化角色项 + 系统语言识别 | 已通过 | `appearance.language=auto` 为默认值，读取 `NSGlobalDomain` 的 `AppleLanguages`（避免 `NSLocale.preferredLanguages` 带入应用级语言覆盖），无法读取时回退 `NSLocale.systemLocale`；匹配 zh-CN/en-US，无法匹配时回退 zh-CN；`internal/menu` 提供“应用/编辑/窗口”原生菜单与角色项，语言切换时重建菜单栏但不重建窗口；`TestResolveLanguageUsesSystemForAuto`、`TestApplicationBuildsLocalizedMenuBar`、`TestApplicationUsesRoleLabels`、`TestApplicationSettingsItem`；真机中英文往返验证关于/设置/隐藏/退出、撤销/重做/剪切/复制/粘贴/全选、最小化/缩放/全屏/关闭均随系统语言更新，设置项可从菜单打开偏好窗口 |

## 8. 设置项

当前设置模型以 `internal/settings/settings.go` 为准；落盘 `config/settings.json`，缺字段补默认值，字段使用 camelCase。设置窗口每个开关/下拉/数字/快捷键控件的路径与回读绑定由 `TestSettingsControlsEmitTheirOwnPath`、`TestSettingsControlsShowLoadedValues` 全量校验，每个可见路径由 `TestSettingsModelAcceptsEveryVisiblePath` 校验可写入。

| ID | 设置组 | 字段 | 迁移要求 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- |
| S-01 | general | `autoStart`、`trayIcon` | `autoStart`/`trayIcon` 已接系统副作用；未接入的 `runAsAdmin` 已移除（§8.1）；`dockIcon` 已移除，Dock 图标跟随偏好设置窗口显隐（见 W-14） | 已通过（产品调整） | `TestSettingsModelAppliesTrayIcon`；真机开发版切换“显示菜单栏图标”后状态即时落盘，重启后保持隐藏，再次开启可恢复图标；默认值、原子写回与重置副作用沿用设置层测试 |
| S-02 | appearance | `theme`、`language` | `auto/light/dark`、`auto/zh-CN/en-US`，语言默认 `auto` | 已通过 | 主题三档已接入并即时应用；`TestSettingsModelAppliesTheme`；语言下拉即时原地更新界面并持久化，`auto` 清除应用级语言覆盖并跟随系统语言，真机中英文往返回读 `settings.json` 正确，`TestSettingsModelAppliesLanguage` |
| S-03 | shortcuts | `openClipboard`、`openPreference` | 两个全局快捷键保留；未接入的 `winV` 已移除（§8.1） | 已通过（产品调整） | 两个快捷键均已读取并注册；“快捷键”页原生录制、清空、非法组合和重复校验可用；`TestSettingsModelAppliesShortcutAndRejectsDuplicate`、`TestSettingsModelRollsBackShortcutWhenRebindFails`；真机 `Alt+K` 改为 `Alt+J` 后立即写盘并重绑 |
| S-04 | clipboard.capture | `text/html/rtf/image/files`、`maxTextMb`、`maxImageMb`、`order` | 采集顺序与上限一致 | 已通过 | `TestCaptureOrderAndTypeSwitches`、`TestSettingsModelAppliesCaptureOrder`；真机设置“采集”页选择纯文本上移/下移后，`settings.json` 即时写入完整五类顺序 |
| S-05 | clipboard.content | `autoPaste`、`copyPlain`、`copyThenHideWindow`、`showOriginalPreview`、删除保护、`autoFavorite`、`sort`、`itemActions`、`itemActionOrder` | 行为语义逐项一致 | 已通过 | 纯文本复制、复制后隐藏、固定记录复用时间/次数、排序方式、单击/双击触发已接入；设置“交互”页可改左键动作、备注自动收藏和“悬停备注条目显示原始内容”，真机写盘验证通过；有备注条目默认显示备注、真实鼠标移入后切回原内容并在移出后恢复备注，`TestHistoryRowTextUsesNoteUntilHover` 覆盖同一逻辑；“删除”页接入普通/收藏/置顶确认与删除保护，列表、右键、悬停和 Delete 键共用同一策略；悬停动作支持勾选与上下排序 |
| S-06 | clipboard.display | `textMaxLines`、`imageMaxHeight`、`fileMaxCount` | 列表展示上限一致：文本标题行数、图片缩略图高度、文件名数量；因 purego 不支持 CGFloat 回调，列表使用统一行高`max(58, 文本行高, 图片高度+24)` | 已通过 | `TestDisplayClampsStoredDensity`、`TestHistoryItemHonorsFileMaxCount`；设置“显示”页可调；真机 `textMaxLines=3` 时 89pt 行高完整显示多行长文本与元数据，`imageMaxHeight=64` 时图片缩略图完整、无裁切，切换文本/图片分类后列表密度一致 |
| S-07 | clipboard.sensitive | `collectSecrets`、`redactSecrets` | 两种策略互斥语义一致 | 已通过 | 设置 UI 可切换；敏感样本与预览脱敏测试 |
| S-08 | clipboard.history | `retention{value,unit}`、`maxCount` | 两项可配置限制 + 内置条件调度 | 已通过（自动化） | 删除自定义清理间隔；启用限制时启动清理并在规则变化后立即执行，之后每 15 分钟执行；两项均停用时不调用清理服务、不创建定时器；`TestRetentionSettingCleansHistoryImmediately`、`TestMaxCountSettingCleansHistoryImmediately`、`TestCleanupWithNoLimitsDoesNotAccessDatabase`、`TestCleanupLoopFollowsEnabledLimits`、`TestUnrelatedSettingsDoNotTriggerCleanup` |
| S-09 | clipboard.search | `defaultFocus`、`clearOnHide` | 面板打开/隐藏行为一致 | 已通过 | 设置“窗口”页提供“打开时聚焦搜索”和“下次打开时清空搜索”，写入 `clipboard.search.*` 后由面板每次 props 重读设置，下一次打开立即使用新值；真机验证表格/搜索框初始焦点、失焦隐藏后保留查询、隐藏后清空查询并恢复完整列表；`TestSettingsModelAppliesPaths`、`TestPanelBehaviorFollowsSettingsOnEveryPropsPass` 覆盖路径映射、回读与即时行为刷新 |
| S-10 | clipboard.window | `position`、`scrollToTopOnOpen`、`selectRangeOnOpen`、`selectCategoryOnOpen`、`selectGroupOnOpen` | 原生窗口等价行为；无运行时消费的 `lightweightMode`/`idleDestroySeconds` 已移除（§8.1） | 已通过（产品调整） | `position` 三档已真机验证；`TestOpenBehavior*`、`TestOpenSelectionAppliesCategoryAndGroup` 覆盖滚动/范围/分类/分组及组合优先级；真机设置“分类=图片 + 分组=全部”冷启动后图片分类选中且仅显示 1 条图片，设置“分组=测试打开”冷启动后分组下拉选中该组且仅显示组内 1 条，分类与分组选择不再互相清空 |
| S-11 | clipboard.preview | `spaceEnabled` | Space 按开关生效；不实现的 hover 预览字段及校验已移除（§8.1），不影响备注悬停显示原文 | 已通过（产品调整） | `TestSpacePreviewHonorsSetting`；设置“预览”页关闭后 Space 不打开预览，开启后立即写入 `clipboard.preview.spaceEnabled`，面板下一次 props 即重读并恢复 Space 预览；`TestPanelBehaviorFollowsSettingsOnEveryPropsPass` 覆盖；设置页改用原生浮层的操作说明，不再暴露迁移计划文案；真机确认 `NSPopover` 按空间出现在面板左侧或右侧，Esc/Space 返回并保持当前行选中；短 HTML 预览按内容收缩至 406×158，长内容滚动到底完整可见，切换条目后回到顶部；默认字体为系统无衬线字体，保留粗体、显式颜色与列表，深色模式下默认深色墨水自动切换语义正文色；`TestAttributedPreviewKeepsNativeDefaultFont`、`TestAttributedPreviewPreservesExplicitColor`、`TestPreviewInkNeedsLabelForDefaultContrast`、`TestPreviewSizeFollowsTextHeight` 覆盖；悬停预览不实现的原因见 `03-migration-plan.md` Phase 4 |
| S-12 | clipboard.feedback | `copySound` | 音效开关一致 | 已通过 | 设置“通用”页开关即时写入 `settings.json`；开启后监听/手动读取入库成功播放系统提示音，关闭后静音；`TestSettingsModelAppliesPaths` 覆盖读写 |
| S-13 | clipboard.filters | `excludedAppIds` | bundle id 过滤一致 | 已通过 | `TestDefaultExcludedAppsAreNotCaptured` 校验默认排除密码类应用且普通来源照常入库；`TestIgnoredAppIsNotCaptured`、`TestSettingsModelAppliesExcludedAppsAndDeletesRemoved`、`TestSettingsModelAddSourceAppExcludesIt` 覆盖自定义排除、取消与来源应用清理 |
| S-14 | onboarding | `completed`、`lastStep`、`legacyImport{checked,imported,importTypes,importedAt}` | 移除引导/导入设置 | 不纳入 macOS 首版（产品调整） | `Onboarding` 结构与设置字段已移除 |
| S-15 | update | 已移除 | 无独立更新配置，更新行为由 NSGo/Sparkle 管理（§8.1） | 已移除（产品调整） | 更新检查入口保留，正式 appcast 与签名升级验收仍见 W-20 |
| S-16 | 默认值 | `settings.Default()` | 缺字段时补当前默认值 | 已通过 | `TestLoadKeepsDefaultsForMissingFields`、`TestOpenFallsBackToDefaultsForUnreadableSettings`、`TestSettingsJSONRoundTripsPublishedCamelCase`：缺字段、空文件、损坏 JSON、类型错误和非法枚举均回退完整默认值，camelCase 编码可无损回环 |
| S-17 | 写入语义 | 对象深合并、数组替换、`.tmp` + rename | patch 语义与原子写一致 | 已通过 | `TestUpdateDeepMergesObjectsAndReplacesArrays`、`TestWriteIsAtomicAndReadable`、`TestConcurrentUpdatesKeepCompleteSettings`；并发 patch 不丢字段，残留 `.tmp` 不影响主文件读取，原子替换后可读 |
| S-18 | 跨字段校验 | 快捷键不重复、`selectGroupOnOpen` 编码合法 | 错误可定位且不破坏旧值 | 已通过 | `TestUpdateRejectsInvalidValuesWithoutChangingSettings`、`TestUpdateRejectsDuplicateShortcuts`：非法枚举、快捷键冲突和非法打开分组均拒绝，内存快照与磁盘文件保持原值 |
| S-19 | 重置 | `reset_settings` | 恢复默认但保留必要身份/数据；设置窗口底部确认后立即生效 | 已通过 | `TestResetSettingsRestoresDefaultsAndSideEffects`：默认值、自启、主题、语言、快捷键、列表刷新均恢复；清理调度随 Store 的默认无限制策略停止；数据目录与历史不受影响；Dock 图标不再由设置控制 |

### 8.1 删除未接入配置字段（2026-11-20）

根据偏好设置审查后的明确取舍，删除以下 12 个没有运行时消费、也没有设置控件的字段：

| 配置路径 | 删除原因 |
| --- | --- |
| `general.runAsAdmin` | Windows 旧设置，当前 macOS 架构未接入 |
| `shortcuts.winV` | Windows 旧快捷键设置，当前 macOS 架构未接入 |
| `clipboard.window.lightweightMode` | 原生面板没有轻量模式分支 |
| `clipboard.window.idleDestroySeconds` | 原生面板没有空闲销毁调度 |
| `clipboard.preview.hoverEnabled` | 不实现通用悬停预览，保留 Space 预览 |
| `clipboard.preview.hoverDelayMs` | 没有悬停预览调度，专属校验一并删除 |
| `update.autoCheck` | 未传入更新器，自动检查应由 NSGo/Sparkle 管理 |
| `update.frequency` | 没有对应调度，专属枚举与校验一并删除 |
| `update.includeBeta` | 未接入 Beta 发布渠道 |
| `update.includeNightly` | 未接入 Nightly 发布渠道 |
| `update.lastCheckedAt` | 没有更新检查时间写回 |
| `update.skippedVersion` | 没有接入更新器的版本过滤 |

整个 `Settings.Update` 组及专属类型、默认值和指针复制辅助函数一并移除；手动检查更新入口、Space 预览、备注悬停显示原文及其他有效设置不变。

2.0.0 不处理旧配置或旧动作值，不提供迁移或回滚到旧版的逻辑。

验证：`gofmt -l .` 无输出，`CGO_ENABLED=0 go vet ./...`、`CGO_ENABLED=0 go test -count=1 ./...`、`CGO_ENABLED=0 go build ./...` 通过。本项未修改视图或窗口行为，未做真机交互回归。

## 8.2 进一步精简偏好设置（2026-11-21）

以下功能在 2.0.0 基线中移除。

1. 中键：删除事件桥接、业务动作、接口、设置状态、校验、控件及专属文案；鼠标左键、右键和滚轮行为保留。

2. 文件路径粘贴：删除开关、菜单、快捷键和悬停动作；所有文件写回复用原生文件对象。

3. 纯文本粘贴：删除开关及所有强制纯文本粘贴入口；普通粘贴保留 HTML/RTF 表示。独立的纯文本复制偏好、复制菜单和 `⌥⌘C` 保留，仅在复制路径生效。

4. 复用排序：删除 `updateOnReuse` 开关；成功复制或准备粘贴后统一累加使用次数并刷新 `updated_at`，失败不更新。最近使用按 `updated_at` 排序，创建时间与使用次数排序各自保留原语义；备注/收藏/置顶/分组仍不刷新复用时间。

5. 自动清理间隔：删除可配置间隔，内部固定为 15 分钟（最短保留时长 1 小时，过期记录最多等待一个周期）。只在有效保留时长或正数上限启用时执行；启动、规则变更立即执行，后续按内置周期执行。全部禁用时停止并释放定时器，不调用清理服务或访问数据库，仅保留等待设置事件的观察器。规则仅启用一项时，数据库仅执行对应条件；收藏/置顶保护及被删图片清理保持不变。清理串行执行并复用应用取消上下文，退出等待清理循环结束后再关闭数据库；移除重复的 UI 清理回调。

6. 日志保留期限：删除配置字段、默认值、校验、持久化路径和 UI；移除运行时保留天数状态与调整 API。日志系统内部固定保留 14 天，启动先清理一次，运行期间每小时清理过期轮转/崩溃日志，并保留最多 20 个归档的容量上限；原 5 MB 轮转压缩、日志级别、脱敏、目录权限及打开目录入口保留。活动主日志仍按大小轮转，不按年龄直接删除。

验证：逐项执行相关 Go 包测试、`CGO_ENABLED=0 go vet ./...` 与 `CGO_ENABLED=0 go build ./...` 后再进入下一项。全部完成后，`gofmt -l .` 无输出，`git diff --check`、`CGO_ENABLED=0 go test -count=1 ./...`、`go test -race ./...`、`make check`、`make build-dev` 通过；`nsgo test cmd/ecopaste` 成功但没有独立 NSGo 测试用例（0 passed/failed/skipped）。开发版真机核对通用/界面/高级页面及滚动布局：六项配置和两个粘贴动作不再展示，纯文本复制、排序方式、保留时长、最大记录数、日志级别与打开日志目录入口保留。数据行为通过临时目录中的设置与备份、复用排序、清理生命周期、文件原生写回及日志保留测试验证；未执行真实目标应用粘贴、真实中键鼠标硬件与真实崩溃回归，不以 UI 展示验证替代这些交互验收。

## 8.3 偏好设置分类重组（2026-11-21）

设置导航按用户任务重组为七类，原有设置字段、存储路径与运行行为不变：

| 分类 | 内容 |
| --- | --- |
| 通用 | 登录启动、菜单栏图标、全局快捷键 |
| 外观 | 主题、语言、文本/图片/文件列表展示密度 |
| 剪贴板 | 采集类型、大小限制、表示优先级、来源应用过滤 |
| 交互 | 面板打开位置与初始选择、搜索、复制/粘贴操作、预览、悬停动作 |
| 数据 | 排序、历史保留和数量、敏感内容策略、删除规则、数据目录与备份恢复 |
| 系统 | macOS 权限、诊断信息、日志级别与日志目录 |
| 关于 | 版本、更新检查、项目链接、设置重置 |

原先混在“通用/界面/高级”的设置按功能场景拆开；应用过滤与采集规则合并到“剪贴板”，历史生命周期及存储/备份归入“数据”。导航仍使用统一的七个图标按钮布局，并为中英文提供一致分类名。`TestEveryScalarSettingBelongsToOneCategory` 覆盖全部原生标量字段的唯一分类，`TestSettingsPagesMatchNavigation` 检查导航和页面一一对应；设置字段的读写路径不变。

## 9. 2.0.0 数据契约

2.0.0 开发包已在独立临时目录完成首次启动与重启，SQLite 完整性检查通过，未生成 SQLx 元数据，也未处理旧版备份暂存目录。开发包与正式身份构建的应用版本和构建版本均为 `2.0.0`。签名、公证与正式更新链路仍需发布验收。

| ID | 优先级 | 契约 | 现有定义 | 迁移要求 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| X-01 | P0 | SQLite | `internal/database/schema.sql` | `user_version=1`；去除未使用的分组与平台字段 | 已通过 | 仓储、备份快照和恢复测试 |
| X-02 | P0 | `updated_at` 语义 | 最近一次复用时间 | 备注/收藏/置顶/分组不刷新 | 已通过 | `TestMetadataUpdatesKeepUpdatedAt` |
| X-03 | P0 | `content_hash` | `blake3("<kind>:<content>")` | 字节级一致 | 已通过 | `TestHashMatchesPublishedFormat` |
| X-04 | P0 | 图片文件名 | `<blake3>.png` 分片 | 路径与命名一致 | 已通过 | `TestImageStoreUsesPublishedBLAKE3NameAndShard`：BLAKE3 空输入向量、文件名 `<digest>.png`、`origin/<digest[:2]>/` 路径符合当前布局 |
| X-05 | P0 | `settings.json` | camelCase、缺字段默认值 | 仅维护当前偏好和 patch 语义 | 已通过 | 设置默认值、校验、并发写入与原子持久化测试 |
| X-06 | P0 | 数据目录 | `storage.json` 锚点 | 当前默认/自定义目录与 dev/prod 隔离 | 已通过 | `TestResolveReadsCustomManifest`、`TestSetRootPointsManifestAtCustomDir`、`TestResolveFallsBackFromBrokenManifest`、`TestResolveRejectsForeignIdentity` |
| X-07 | P0 | `.ecopastebak` | ZIP manifest `formatVersion=2`；加密标识 `ECOPASTEBAK2` | 2.0.0 明文和密码备份；仅当前格式 | 已通过（自动化） | 导入前校验归档格式与数据库版本；加密头只存 salt 与 nonce，当前格式固定使用 Argon2id 与 XChaCha20-Poly1305 |
| X-08 | P0 | 文件关联 | `.ecopastebak` | 同扩展名/UTI | 已通过 | `nsgo.yaml` 声明 `com.ayangweb.eco-paste.backup`，构建产物 Info.plist 校验 `CFBundleDocumentTypes`；真机运行中/冷启动 `open -a` 均进入导入确认，导入后重启校验 DB |
| X-09 | P0 | Bundle ID | `com.ayangweb.eco-paste` | 正式版一致；开发版隔离 | 迁移中 | `make dev`/`make build-dev` 使用 `com.ayangweb.eco-paste.dev`；开发构建误用正式 ID 时启动退出；开发 bundle ID 在 `ECOPASTE_MODE` 之前映射到 `dev` 数据根，避免覆写正式数据；`lsof` 实测开发实例只访问 `dev/db`、`dev/logs`、`dev/state`；正式版安装与登录项待签名包验收 |

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

## 11. 性能与资源生命周期复核（2026-11-21）

完整依据、定时任务清单与基准见 [08-performance-audit.md](./08-performance-audit.md)。本轮没有改变既有功能的产品范围，也不以本轮局部回归代替所有既有人工验收。

| 关联项 | 本轮结果 | 验证与边界 |
| --- | --- | --- |
| C-01 / C-18 / C-20 | 暂停不创建/运行采样 ticker，恢复事件重新启用；手动读取保留且不再重复发送刷新信号 | watcher 生命周期测试与 race 测试；物理复制/粘贴矩阵未重跑 |
| D-03 / D-05 / U-01 / U-07 | 隐藏不读历史；快速搜索取消旧查询；搜索与筛选合并；行展示仅在数据/语言变化时重算 | `TestHiddenHistoryDefersRefreshUntilOpen`、`TestHistoryFollowsWindowVisibilityWithoutNativeNotifications`、`TestFilterConsumesPendingSearchWithoutDuplicateOrMixedPages`、`TestHistoryRowsAreMemoizedAcrossUnrelatedProps`；150 条隔离夹具真机验证首次加载、快速搜索与预览 |
| C-05 / D-21 | 已缓存缩略图不重复解码/缩放/编码；缺失仍重建 | `TestCachedThumbnailSkipsImageDecoding`、`TestImageStoreRecreatesMissingThumbnail`；重复 1600×1000 PNG 的局部基准约 30ms→0.08ms，不代表首次解码或整应用速度 |
| W-03 / W-05 / W-07 / S-05 | 隐藏/销毁取消加载与点击延时；未启用单击不安排任务；释放设置与面板的原生引用，复用 hover tracking area，cell 销毁清理 layout | 原生 lifecycle/tracking 测试；真机设置值 3→4→3、关闭重开通过；Carbon 全局热键、真实拖放与目标应用粘贴仍需人工复核；拖拽兜底保留并在原生结束时立即取消 |
| A-12 / A-13 / A-14 | 保留原生拖放与既有检测兜底；结束/隐藏/关闭统一释放拖拽资源 | `TestShouldHideAfterDrag`；本轮自动 drag 未能确认真实拖放，不更新为新一轮真机通过 |

中断后复核补齐 runtime 上下文捕获的线程/owner 边界，以及服务不可用时首次显示的错误状态；增加设置 worker、服务缺失和原生延时任务取消测试，完整 Go / race / NSGo 检查与开发构建重新通过。隔离 150 条数据的加载、搜索、预览和设置重开回归通过；真实拖放与粘贴仍按表中边界保留。详见性能排查文档的“中断后的复核与补充”。

## 12. 验收记录规则

- 每完成一项，把状态改为 `已通过`，并在验证列补充实际命令、截图路径或手动步骤编号。
- 重要的产品取舍必须写明理由，不能只写“已通过”。
- 新增功能先加入本表，再实现，避免 checklist 与代码脱节。
- 退役 Tauri 前，P0/P1 不允许存在 `待迁移` 或 `迁移中`；P2 例外必须记录。
