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
| C-18 | P1 | 监听暂停 | `WatcherPause`、托盘 | 暂停期间跳过入库，watcher 不退出 | 迁移中 | 托盘菜单已接；待真机验证 |
| C-19 | P2 | 复制音效 | `clipboard/sound.rs` | `AVAudioPlayer` 或 Go 音频库 | 待迁移 | 开关、静音、缺失文件 |
| C-20 | P1 | 手动读取剪贴板 | `read_clipboard` | 与自动采集共用同一条管线 | 待迁移 | 空剪贴板、重复内容 |

## 2. 数据、历史与存储

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| D-01 | P0 | SQLite 主库 | `db/init.rs`、`db/path.rs` | `modernc.org/sqlite`，同路径同 schema | 已通过 | `go test ./internal/database`：新库创建、旧时间格式读取 |
| D-02 | P0 | Migration 兼容 | `_sqlx_migrations`、`0001_init.sql` | 识别旧 migration，不重复建表 | 已通过 | 检测 `clipboard_items` 后跳过建表；待补 `_sqlx_migrations` 显式记录 |
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
| D-17 | P0 | 启动清理 | `clipboard::init` | 启动先跑一次，再按间隔 | 迁移中 | 启动与周期调度已接；待真机长运行验证 |
| D-18 | P0 | 清空记录 | `clear_items` | 可选保留收藏/置顶 | 迁移中 | 托盘原生对话框已接；`TestClearHistoryOptionallyKeepsFavoritesAndPinned` |
| D-19 | P0 | 数据根解析 | `core/paths.rs` | `<app_local_data>/<dev\|prod>` + `storage.json` | 已通过 | `internal/storage` 测试覆盖默认根、锚点与自定义目录 |
| D-20 | P0 | 自定义数据目录 | `commands/storage.rs` | 迁移 db/resources/config/state 并热切换 | 待迁移 | 切换、失败回滚、重启 |
| D-21 | P0 | 资源目录 | `core/paths.rs` | 同目录结构，图片按 hash 分片 | 已通过 | `TestImageStoreIsIdempotent`；真机图片落盘 |
| D-22 | P1 | 存储占用 | `get_storage_usage` | total/database/resources/settings 分项 | 待迁移 | 与 `du` 误差可解释 |
| D-23 | P1 | 缓存清理 | `clean_resource_cache` | 删除未被 DB 引用的图片/图标 | 待迁移 | 孤儿文件清理、引用保留 |
| D-24 | P1 | 旧版数据导入 | `commands/onboarding.rs` | 读 v0.6 `history` + `images/` | 待迁移 | 普通/收藏选择性导入 |
| D-25 | P1 | 来源应用注册表 | `db/apps.rs`、`apps_registry.rs` | 列表、全部应用、从路径添加、清理未引用 | 待迁移 | 增删与去重 |
| D-26 | P1 | 文件图标缓存 | `db/file_icons.rs`、`file_icon_store.rs` | `NSWorkspace` 图标 + cache key | 待迁移 | 不同扩展名/同扩展名 |
| D-27 | P1 | 备份入口 | `commands/backup.rs` | 导出、检查、导入、pending slot | 待迁移 | 明文/加密/取消/恢复 |
| D-28 | P1 | 打开数据目录 | `commands/storage.rs` | `NSWorkspace` 打开目录 | 待迁移 | 默认/自定义目录 |

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
| P-08 | P1 | 文件条目 | `commands/clipboard.rs` | 路径、名称、目录/图片标记、存在性、大小 | 待迁移 | 单文件/多文件/丢失文件 |
| P-09 | P1 | 文件预览模式 | `FilesPreviewKind` | 单图片走图片预览，其余列表 | 待迁移 | 三种组合 |
| P-10 | P1 | 预览窗口 | `window/preview.rs`、`pages/Preview` | 面板内预览视图，Space/Esc，文本/图片/文件 | 已通过 | 真机文本/图片预览与 Esc 返回；产品调整记录在迁移计划 |
| P-11 | P2 | HTML/RTF 富文本预览 | 当前只用纯文本摘要 | `NSAttributedString` + `NSTextView`（增强项） | 待迁移 | 与旧版行为差异需记录 |

## 4. 历史复用动作

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| A-01 | P0 | 写回纯文本 | `clipboard/write.rs` | `NSPasteboard` plain text | 已通过 | 真机选中列表项 Enter 写入，`pbpaste` 校验一致 |
| A-02 | P0 | 写回 HTML | `write.rs` | plain + `public.html` 同一 item | 已通过 | 真机复制 `<b>rich hello</b>` 后剪贴板含 HTML + plain，`pbpaste` 得到 `rich hello`，DB 不新增回环 |
| A-03 | P0 | 写回 RTF | `write.rs` | plain + `public.rtf` 同一 item | 迁移中 | 同一 pasteboard item 与回环登记已实现；待真机 Pages/Word 验证 |
| A-04 | P0 | 写回图片 | `write.rs` | `public.png` + TIFF（按需） | 迁移中 | PNG 写回已实现；待真机验证 |
| A-05 | P0 | 写回文件 | `write.rs` | `public.file-url` 多文件 | 迁移中 | `NSURL` 多文件写回已实现；待真机验证 |
| A-06 | P0 | 粘贴 | `paste_clipboard_item` | 写回 + 让出 key + `CGEvent` ⌘V | 迁移中 | 回车/双击已接 `CGEvent`；权限拒绝路径已验证，待授权后真机验收 |
| A-07 | P0 | 纯文本粘贴 | `paste_plain` | 文本只写 plain；HTML/RTF 带 plain 回退 | 迁移中 | 菜单“纯文本粘贴/复制为纯文本”已真机验证只写 plain；自动按键待辅助功能授权 |
| A-08 | P1 | 文件路径粘贴 | `paste_files_as_path` | 文件记录写回换行路径文本 | 已通过 | `TestBuildWritePreservesRepresentations`；真机“粘贴为路径”与 `pasteFilesAsPath=true` 正常粘贴均写入换行路径，未新增回环 |
| A-09 | P1 | 打开链接/邮件 | `open_clipboard_item_link` | `NSWorkspace.open` | 已通过 | `TestOpenDispatchesBySubKind`；URL/路径条目右键菜单显示对应动作 |
| A-10 | P1 | Finder 显示 | `reveal_clipboard_item` | `NSWorkspace.activateFileViewerSelecting` | 已通过 | `TestRevealFilesAndPath`；真机对路径条目执行后 Finder 选中目标目录 |
| A-11 | P1 | 图片另存 | `save_clipboard_image_to_file` | `NSSavePanel`，默认文件名规则 | 已通过 | `TestSaveImageWritesChosenFile`；真机保存 3456×2234 PNG，文件头与尺寸校验通过 |
| A-12 | P1 | 拖出文件 | `drag_out/macos.rs` | `NSDraggingSession` + file URL | 待迁移 | Finder/浏览器/聊天工具 |
| A-13 | P1 | 拖出图片 | `drag_out/macos.rs` | `NSDraggingSession` + PNG | 待迁移 | 同上 |
| A-14 | P1 | 拖出文本 | `drag_out/macos.rs` | plain/HTML/RTF 多表示 | 待迁移 | 富文本目标 |
| A-15 | P1 | 备注编辑 | `NoteModal` | 原生备注弹窗，256 字符上限 | 已通过 | `TestEditNoteNormalizesAndAutoFavorites`、`TestEditNoteCancelAndClear`、`TestEditNoteLimitsLength`；保存/取消/空值/自动收藏已覆盖 |
| A-16 | P1 | 移动到分组 | 右键子菜单 | `NSMenu` 动态分组列表与当前分组勾选 | 已通过 | `TestMoveToGroupAndBackToUngrouped`；真机移动到“工作”后分组筛选仅显示 1 条 |
| A-17 | P0 | 删除条目 | `delete_clipboard_item` | 删除 DB 行 + 图片文件 | 已通过 | 右键/Delete 确认；`TestDeleteRemovesImageFile` |
| A-18 | P1 | 自动粘贴 | `Content.auto_paste` | 单击/双击、粘贴/复制、禁用 | 迁移中 | 双击/回车与复制/禁用已接；单击触发待补 |
| A-19 | P1 | 中键动作 | `Content.middle_click` | 粘贴/纯文本粘贴/复制/纯文本复制/禁用 | 待迁移 | 真实鼠标 |
| A-20 | P1 | 悬停快捷动作 | `Content.item_actions` | 可配置、可排序 | 待迁移 | 配置变更即时生效 |
| A-21 | P1 | 复制后隐藏 | `copyThenHideWindow` | 复制后按设置隐藏，固定窗口时保持可见 | 已通过 | `TestHideAfterActionRespectsPinnedPanel`；真机固定后复制面板保持，取消固定后复制面板隐藏 |

## 5. 窗口与系统集成

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| W-01 | P0 | 剪贴板面板 | `window/macos.rs` | `NSPanel`，非激活、可 key、置顶、无边框、透明 | 待迁移 | 全屏/多 Space/多屏 |
| W-02 | P0 | 窗口定位 | `window/position.rs` | 跟随鼠标/居中/记住位置 | 已通过 | 设置“面板打开位置”三档；真机 `osascript` 读取窗口 frame：居中 `654,217`、跟随光标 `300,393`（光标 `300,400`）、记住位置恢复 `654,217` |
| W-03 | P0 | 失焦隐藏 | `window/macos.rs` | 自定义 resign-key 隐藏 + modal/固定窗口期间暂停 | 迁移中 | 真机切换 Finder 后隐藏；系统另存/备注弹窗期间保持；固定窗口保持可见；拖拽保护待补 |
| W-04 | P0 | 固定窗口 | `set_clipboard_window_pinned` | 固定不隐藏；粘贴时 resign/restore key | 迁移中 | 原生按钮与 `P` 键切换已接；真机固定后失焦/复制均保持可见；粘贴 key 让出/恢复待授权后验收 |
| W-05 | P1 | 窗口生命周期 | `window/lifecycle/*` | 原生窗口状态机，无 WebView 重建 | 待迁移 | 冷启/热显/隐藏/退出 |
| W-06 | P1 | 窗口几何持久化 | `window/state.rs` | 同 `state/window-state.json` | 已通过（格式调整） | 隐藏时写入 `clipboard` 帧并重启恢复；旧版存物理像素、左上原点，新版存 AppKit points、左下原点，JSON 字段与路径不变；旧坐标不在任一屏幕时回退到光标所在屏幕居中 |
| W-07 | P1 | 偏好窗口 | `window/mod.rs` | 原生 `NSTabView` 设置窗口 | 迁移中 | 通用/采集/历史与隐私已可用；深链与高亮待补 |
| W-08 | P1 | 更新窗口 | `window/mod.rs` | Sparkle 交互或独立窗口 | 待迁移 | 检查/下载/安装 |
| W-09 | P1 | 引导窗口 | `window/mod.rs` | 6 步原生流程 | 待迁移 | 回退/恢复/完成 |
| W-10 | P0 | 托盘 | `tray/mod.rs` | `NSStatusItem` + `NSMenu`：偏好、暂停、开源、更新、版本、重启、退出 | 迁移中 | 显示/暂停/设置/清除/开源/版本/重启/退出已接；更新项待更新模块完成后补 |
| W-11 | P0 | 全局快捷键 | `shortcut/mod.rs` | `desktop.RegisterHotKey` | 迁移中 | `Alt+C`、`Alt+X` 已注册；冲突与热更新待验证 |
| W-12 | P1 | 快捷键录制 | 前端录制器 | 原生 `NSTextField` 捕获 | 待迁移 | 组合键/冲突/清除 |
| W-13 | P1 | 开机自启 | `autostart/macos.rs` | `SMAppService` | 待迁移 | 开启/关闭/重启 |
| W-14 | P1 | Dock 图标显隐 | `show_taskbar_icon` | `NSApplication.setActivationPolicy` | 待迁移 | 显示/隐藏/重启 |
| W-15 | P1 | 单实例 | `tauri-plugin-single-instance` | macOS bundle 单实例 + `OnReopen` | 待迁移 | 二次启动/文件打开 |
| W-16 | P1 | 文件关联 | `tauri.conf.json`、`backup/*` | `nsgo.yaml file_types` + `OnOpenFiles` | 待迁移 | 双击/拖到 Dock |
| W-17 | P1 | 原生右键菜单 | `menu/clipboard_item.rs` | `NSMenu` 按内容动态生成动作 | 已通过 | URL/图片/路径菜单与分组子菜单已真机验证；`⌘C`/`⌘V`/`⌥⌘C`/`⌥⌘V` 与标准编辑菜单复制/粘贴已真机验证 |
| W-18 | P1 | 权限 | `PermissionControl` | 辅助功能/完全磁盘访问检测与跳转 | 迁移中 | 辅助功能检测、原生说明弹窗与系统设置跳转已接；FDA 待补 |
| W-19 | P2 | 防默认行为 | `core/prevent_default.rs` | 无 WebView，不需要 | 不纳入 macOS 首版 | 确认无 WebView 入口 |
| W-20 | P1 | 更新 | `update.rs` | NSGo updater + Sparkle appcast | 待迁移 | 测试源真实升级 |
| W-21 | P1 | 日志 | `tauri-plugin-log` | `log/slog` + 轮转，敏感信息脱敏 | 待迁移 | 路径/轮转/脱敏 |

## 6. UI 页面与交互

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| U-01 | P0 | Clipboard 主页面 | `legacy/frontend/src/pages/Clipboard` | AppKit 搜索/分类/分组/列表/卡片/Footer | 迁移中 | 原生搜索、分类、自定义分组筛选、列表、Footer、实时刷新与复制已真机验证；卡片待补 |
| U-02 | P1 | Preview 页面 | `legacy/frontend/src/pages/Preview` | 面板内原生预览 | 已通过 | 真机 Space/Esc；`TestPreviewRedactsSensitiveText` |
| U-03 | P1 | Preference 页面 | `legacy/frontend/src/pages/Preference` | 原生 Tab/表单/搜索/高亮 | 待迁移 | 7 个 Tab 全项 |
| U-04 | P1 | Onboarding 页面 | `legacy/frontend/src/pages/Onboarding` | 原生 6 步流程 | 待迁移 | 全流程/回退 |
| U-05 | P1 | Update 页面 | `legacy/frontend/src/pages/Update` | 原生更新状态/进度 | 待迁移 | 各状态 |
| U-06 | P2 | ContextMenu 页面 | `legacy/frontend/src/pages/ContextMenu` | macOS 使用 `NSMenu`，不迁移 WebView 菜单窗 | 参考 | 对比菜单结构 |
| U-07 | P0 | 列表虚拟滚动 | `react-virtuoso` | `NSTableView` 原生虚拟化 | 迁移中 | `NSTableView` 已启用原生复用与 80 条分页；待 10k 条压测 |
| U-08 | P1 | 键盘导航 | `legacy/frontend/src/hooks` 的键盘逻辑 | 上下/Page/Home/End/Enter/Esc | 迁移中 | 真机 ↑/↓、Home/End、⌘F、⌘C、⌥⌘C、⌘V、⌥⌘V 通过；PageUp/PageDown 已兼容专用键与 Fn+↑/↓，自动化环境未产生对应按键事件 |
| U-09 | P1 | 主题 | antd token + `light/dark` | AppKit 语义色 + `NSAppearance` | 待迁移 | 跟随系统/强制 |
| U-10 | P1 | 动画 | `motion` | AppKit 动画/`NSAnimationContext` | 待迁移 | 面板出现/预览连接线 |
| U-11 | P2 | 滚动条 | `overlayscrollbars` | 系统 overlay scroller | 待迁移 | 深浅色 |
| U-12 | P2 | 诊断 | `pages/Preference` 诊断区 | 原生诊断面板/复制信息 | 待迁移 | 数据目录/版本/权限 |

## 7. 国际化与文案

| ID | 优先级 | 功能 | 现有实现 | NSGo/AppKit 目标 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| I-01 | P1 | 前端文案 | `legacy/frontend/src/locales/zh-CN`、`en-US` | Go i18n 目录，复用文案内容 | 待迁移 | 逐 key 对比 |
| I-02 | P1 | Rust 侧文案 | `src-tauri/src/i18n/` | Go 文案目录 | 待迁移 | 托盘/菜单/toast |
| I-03 | P1 | 语言切换 | `appearance.language` | 原生菜单/设置切换，立即生效 | 待迁移 | 重启/即时 |
| I-04 | P2 | 系统本地化 | 部分系统菜单 | AppKit 本地化角色项 | 待迁移 | 中英文系统 |

## 8. 设置项

设置模型以 `legacy/backend/src/settings/model.rs` 为准，落盘 `config/settings.json`，全部 `serde(default)`、camelCase。

| ID | 设置组 | 字段 | 迁移要求 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- |
| S-01 | general | `autoStart`、`runAsAdmin`、`trayIcon`、`dockIcon` | 保留字段；`runAsAdmin` 仅 Windows 参考 | 待迁移 | 旧文件回环/系统行为 |
| S-02 | appearance | `theme`、`language` | `auto/light/dark`、`zh-CN/en-US` | 待迁移 | 即时切换/重启 |
| S-03 | shortcuts | `openClipboard`、`openPreference`、`winV` | 保留；`winV` 仅 Windows 参考 | 迁移中 | 两个快捷键均已读取并注册；编辑 UI 待补 |
| S-04 | clipboard.capture | `text/html/rtf/image/files`、`maxTextMb`、`maxImageMb`、`order` | 采集顺序与上限一致 | 迁移中 | 五个开关与两个上限已接设置 UI；顺序编辑待补 |
| S-05 | clipboard.content | `autoPaste`、`middleClick`、`copyPlain`、`copyThenHideWindow`、`pastePlain`、`pasteFilesAsPath`、`showOriginalPreview`、删除保护、`autoFavorite`、`updateOnReuse`、`sort`、`itemActions`、`itemActionOrder` | 行为语义逐项一致 | 迁移中 | 纯文本复制/粘贴、文件路径粘贴、复制后隐藏、复用排序、排序方式已接入；中键与悬停动作待补 |
| S-06 | clipboard.display | `textMaxLines`、`imageMaxHeight`、`fileMaxCount` | 列表展示上限一致 | 待迁移 | 边界值 |
| S-07 | clipboard.sensitive | `collectSecrets`、`redactSecrets` | 两种策略互斥语义一致 | 已通过 | 设置 UI 可切换；敏感样本与预览脱敏测试 |
| S-08 | clipboard.history | `retention{value,unit}`、`maxCount`、`cleanupIntervalHours` | 清理规则一致 | 已通过 | 设置 UI 可调；`TestCleanupHistoryByRetentionKeepsProtectedItems`、`TestCleanupHistoryByMaxCount` |
| S-09 | clipboard.search | `defaultFocus`、`clearOnHide` | 面板打开/隐藏行为一致 | 已通过 | 真机验证表格/搜索框初始焦点、失焦隐藏后保留查询、隐藏后清空查询并恢复完整列表 |
| S-10 | clipboard.window | `position`、`scrollToTopOnOpen`、`selectRangeOnOpen`、`selectCategoryOnOpen`、`selectGroupOnOpen`、`lightweightMode`、`idleDestroySeconds` | 原生窗口等价行为；无 WebView 时重新解释轻量化 | 迁移中 | `position` 三档已接入设置并真机验证；打开时滚动/选中策略、轻量化与空闲销毁待补 |
| S-11 | clipboard.preview | `hoverEnabled`、`hoverDelayMs`、`spaceEnabled` | Space 已接；hover 暂不实现 | 迁移中 | 延迟边界 |
| S-12 | clipboard.feedback | `copySound` | 音效开关一致 | 待迁移 | 开/关 |
| S-13 | clipboard.filters | `excludedAppIds` | bundle id 过滤一致 | 待迁移 | 默认排除/自定义 |
| S-14 | onboarding | `completed`、`lastStep`、`legacyImport{checked,imported,importTypes,importedAt}` | 首次启动与导入状态一致 | 待迁移 | 中断/恢复/完成 |
| S-15 | update | `autoCheck`、`frequency`、`includeBeta`、`includeNightly`、`lastCheckedAt`、`skippedVersion` | 渠道与跳过逻辑一致 | 待迁移 | 测试 appcast |
| S-16 | 默认值 | 所有 `serde(default)` | 缺字段时行为与旧版一致 | 待迁移 | 空文件/缺字段 |
| S-17 | 写入语义 | 对象深合并、数组替换、`.tmp` + rename | patch 语义与原子写一致 | 待迁移 | 并发/崩溃模拟 |
| S-18 | 跨字段校验 | 快捷键不重复、`selectGroupOnOpen` 编码合法 | 错误可定位且不破坏旧值 | 待迁移 | 非法输入 |
| S-19 | 重置 | `reset_settings` | 恢复默认但保留必要身份/数据 | 待迁移 | 重置后重启 |

## 9. 数据契约与兼容

| ID | 优先级 | 契约 | 现有定义 | 迁移要求 | 状态 | 验证方式 |
| --- | --- | --- | --- | --- | --- | --- |
| X-01 | P0 | SQLite schema | `migrations/0001_init.sql` | 表、列、索引、触发器一致 | 已通过 | `internal/database/schema.sql` 直接复制已发布 migration |
| X-02 | P0 | `updated_at` 语义 | 最近一次复用时间 | 备注/收藏/置顶/分组不刷新 | 已通过 | `TestMetadataUpdatesKeepUpdatedAt` |
| X-03 | P0 | `content_hash` | `blake3("<kind>:<content>")` | 字节级一致 | 已通过 | `TestHashMatchesPublishedFormat` |
| X-04 | P0 | 图片文件名 | `<blake3>.png` 分片 | 路径与命名一致 | 待迁移 | 旧数据直接引用 |
| X-05 | P0 | `settings.json` | camelCase、serde default | 字段、默认值、patch 语义一致 | 待迁移 | 旧文件读写回环 |
| X-06 | P0 | 数据目录 | `storage.json` 锚点 | 默认/自定义目录均可解析 | 待迁移 | 目录切换 |
| X-07 | P0 | `.ecopastebak` | ZIP / 自定义加密容器 | 双向读写兼容 | 待迁移 | 旧包互导 |
| X-08 | P0 | 文件关联 | `.ecopastebak` | 同扩展名/UTI | 待迁移 | 双击/拖放 |
| X-09 | P0 | Bundle ID | `com.ayangweb.eco-paste` | 正式版一致；开发版隔离 | 待迁移 | 安装/升级/数据目录 |

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
