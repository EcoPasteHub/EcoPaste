# 目标 Go + nsgo + AppKit 架构

## 1. 架构目标

目标应用是一个单一 Go 进程：

```text
Go 业务核心
  + nsgo 应用壳（窗口/Panel/托盘/快捷键/文件打开/重启）
  + AppKit 原生控件树
  + SQLite
  + 自建 macOS 平台适配
```

硬约束：

- 不使用 WebView。
- 不使用 nsgo 的 `base` / `ui` 作为产品控件。
- 所有用户可见操作控件必须是 AppKit 原生控件。
- 第一阶段只实现 macOS。
- Windows 代码保留在 Rust 现状中作为对照，不在本阶段迁移。
- 旧数据库、旧备份和旧版历史数据尽可能兼容。

## 2. 分层

```text
cmd/ecopaste
  └─ internal/app
       ├─ internal/domain
       ├─ internal/clipboard
       ├─ internal/storage
       ├─ internal/settings
       ├─ internal/backup
       ├─ internal/search
       ├─ internal/update
       └─ internal/macos
            ├─ internal/macos/appkit
            ├─ internal/macos/platform
            └─ internal/ui
```

职责边界：

| 层 | 职责 | 禁止 |
| --- | --- | --- |
| `cmd/ecopaste` | 进程入口、nsgo.Run、全局错误处理 | 业务逻辑 |
| `internal/app` | 生命周期、依赖装配、窗口协调 | 直接画 UI |
| `internal/domain` | 纯模型、值对象、业务规则 | AppKit/SQLite 依赖 |
| `internal/clipboard` | 采集、类型识别、敏感检测、写回、回环抑制 | 直接操作 DB |
| `internal/storage` | SQLite schema、仓储、查询、迁移 | AppKit |
| `internal/search` | 关键词解析、FTS/LIKE 规则 | UI |
| `internal/settings` | 设置模型、默认值、patch、原子写入 | AppKit |
| `internal/backup` | `.ecopastebak` 格式、导入导出 | UI |
| `internal/macos/appkit` | purego/objc 或 cgo 的 AppKit 包装 | 业务规则 |
| `internal/macos/platform` | pasteboard、Workspace、CGEvent、拖拽、权限 | UI 布局 |
| `internal/ui` | 窗口控制器、列表、设置页、菜单 | SQL/文件格式 |
| `internal/update` | 更新协议、Sparkle/Tauri 迁移适配 | 剪贴板业务 |

当前仓库尚未建立 Go 实现骨架；本阶段不创建 `cmd/` 或 `internal/`。未来可按上述结构落地，实际目录名在实现阶段再确定。

## 3. 进程与主线程

### 3.1 AppKit 主线程

- `main` 必须锁定 OS 主线程。
- nsgo 负责启动 AppKit 事件循环。
- 所有 `NSView`、`NSWindow`、`NSPanel`、`NSMenu`、拖拽、responder chain 操作必须在主线程。
- Go 服务可以运行在 goroutine，但 UI 更新必须 `dispatch` 回主线程。

建议：

```text
clipboard watcher goroutine
  -> event channel
  -> app coordinator
  -> main-thread dispatcher
  -> UI controllers
```

### 3.2 事件总线

不复制 Tauri IPC，但保留“域事件”概念：

```text
clipboard.changed
settings.changed
groups.changed
window.visibilityChanged
window.lifecycleChanged
backup.received
update.progress
shortcut.conflict
```

事件由 Go 内部发布，UI 订阅。跨窗口共享状态由 app coordinator 持有，不要让每个窗口各自读数据库。

## 4. 数据与存储

### 4.1 数据库

目标：

- 直接兼容 `clipboard.db`。
- 使用 SQLite WAL。
- 保持 FTS5 trigram。
- 保持记录 ID、`content_hash`、时间字段和资源文件名。

建议将仓储拆为：

- `item_repository`
- `group_repository`
- `app_repository`
- `icon_repository`
- `query_repository`
- `backup_repository`

查询语义必须与当前实现一致：

- 置顶恒前置。
- 3 字符以上 FTS，短词 LIKE。
- 文本列表裁剪完整 content。
- 元数据更新不刷新 `updated_at`。
- 去重命中只累加 `use_count`。

### 4.2 SQLite 驱动决策

第一步不是直接选定驱动，而是做 spike：

1. `modernc.org/sqlite`。
2. `ncruces/go-sqlite3`。
3. 如必要，cgo SQLite 仅作为对照。

通过标准：

- 打开真实旧库。
- 读取 `_sqlx_migrations`。
- FTS5 trigram 可用。
- external content trigger 正常。
- WAL 并发和 busy timeout 正常。
- 备份/复制数据库时一致性可保证。

建议优先纯 Go 方案，避免 cgo 与 nsgo 的零 cgo 架构冲突；但最终选择以 spike 结果为准。

### 4.3 Schema 接管

```text
启动
  -> 只读探测 schema
  -> 识别旧 Rust 库 / 新 Go 库 / 旧版 history / 空库 / 未知
  -> 必要时生成迁移前快照
  -> 建立或更新 Go schema ledger
  -> 打开仓储
```

不自动修复未知 schema，不重放 `0001_init.sql` 到旧库。

### 4.4 资源

保留：

- `resources/clipboard-images/origin`
- `resources/clipboard-images/thumbnails`
- `resources/app-icons`
- `resources/file-icons`

Go 端实现同样的哈希分片和懒生成缩略图。切换数据目录时重新绑定资源根。

## 5. 设置

目标设置模型沿用当前 JSON：

- camelCase。
- 每个字段有默认值。
- 缺字段补默认。
- 深度 patch。
- 临时文件 + rename 原子写。

建议把设置分成：

- `Settings` 聚合。
- `General`、`Appearance`、`Shortcuts`、`Clipboard`、`Update`。
- `Clipboard` 下再分 capture/content/display/sensitive/history/search/window/preview/feedback/filters。

UI 只读写 go 设置服务，不直接解析 JSON。

## 6. 剪贴板服务

### 6.1 端口

建议定义平台端口：

```text
PasteboardReader
PasteboardWriter
PasteboardWatcher
FrontmostAppProvider
AppIconProvider
PasteSimulator
DragOutService
PermissionService
```

业务层只依赖端口，macOS 实现放在 `internal/macos/platform`。

### 6.2 采集管线

```text
watcher
  -> FrontmostAppProvider
  -> excluded app filter
  -> PasteboardReader
  -> Normalizer
       text/html/rtf/files/image
  -> SecretDetector
  -> content hash
  -> WritebackGuard
  -> ItemRepository.Upsert
  -> AppRepository.Upsert
  -> EventBus.Publish(clipboard.changed)
```

必须保留：

- 类型采集顺序。
- 文本必须 plain 非空。
- HTML/RTF 同时写 plain fallback。
- 图片 PNG 直通优先。
- 文件路径列表和 file_types。
- 来源应用的相对稳定 ID。
- 敏感内容和排除应用。

### 6.3 写回

`PasteboardWriter` 需要支持：

- text
- html + plain
- rtf + plain
- image PNG
- files
- files as text

写回前登记 `content_hash`，监听侧消费。

## 7. AppKit UI

### 7.1 Native 根

每个窗口：

```text
nsgo Window/Panel
  -> full-size nsgo container
       -> core.Native{Component: "ecopaste-appkit-root"}
            -> NSView root
                 -> AppKit controls
```

根组件由 EcoPaste 注册，不由 nsgo 组件库提供。

nsgo 只会把 Native 根的原生视图按 width/height autoresizing 加到节点 bounds 中，不负责 first responder、输入法、Tab 顺序或菜单验证。这些必须由 EcoPaste 的 AppKit 控制器显式设置，并在真实窗口中验证。

### 7.2 主剪贴板窗口

目标控件：

- `NSPanel` / `NSWindow` 外壳。
- `NSSearchField`。
- `NSSegmentedControl` 或自定义 AppKit tab bar。
- `NSScrollView` + `NSTableView`（或 `NSCollectionView`）。
- `NSTableCellView` 子类：
  - 文本摘要
  - 图片缩略图
  - 文件列表
  - 来源应用图标
  - 收藏/置顶/备注状态
  - hover 操作按钮
- 底部状态栏、分组按钮、pin。
- `NSMenu` 右键菜单。

### 7.3 列表行为

需要实现：

- 行复用和图片异步加载。
- 选中行、键盘上下移动、Enter 粘贴、Space 预览、Escape 关闭。
- 双击/单击动作按设置执行。
- 多选（如产品需要）。
- 滚动到顶部、分页/增量加载。
- 右键菜单状态勾选、分组子菜单、菜单验证。
- 拖出文件/图片/文本。

### 7.4 偏好设置

用 `NSWindowController` + `NSTabViewController` / `NSSplitViewController`：

- 常规、外观、快捷键、剪贴板、历史、存储、权限、更新、关于。
- 原生 `NSButton`、`NSSwitch`、`NSPopUpButton`、`NSSlider`、`NSTextField`、`NSTableView`。
- 录制快捷键。
- 原生文件选择器 `NSOpenPanel` / `NSSavePanel`。
- 权限状态轮询和系统设置跳转。

现有 `preferenceSchema.ts` 只作为字段和分组参考，不作为 UI 组件结构。

### 7.5 预览

不使用 WebView：

- 文本：`NSTextView` 或 `NSTextField`。
- HTML/RTF：`NSAttributedString` 原生富文本解析；仍需安全处理，禁止远程资源和脚本。
- 图片：`NSImageView`。
- 文件：`NSStackView` + `NSImageView` + `NSTextField`。
- 面板：独立无边框 `NSPanel`，根据锚点计算位置。

### 7.6 菜单与菜单栏

- 列表右键：`NSMenu`。
- 菜单项：paste、paste plain、paste path、copy、copy plain、save image、open link、send email、reveal、favorite、pin、move to group、note、delete。
- 菜单栏：`NSStatusItem` + `NSMenu`，可由 nsgo tray 或 AppKit 直接实现。
- 菜单验证通过 `validateMenuItem:` / `NSMenuItemValidation`。

### 7.7 拖入/拖出

- 拖入 `.ecopastebak`：AppKit `NSDraggingDestination`。
- 拖出：`NSDraggingSource` + `NSPasteboardItem`。
- 图片/文本预览图必须按 point 尺寸显示，不能把像素当 point。

## 8. 窗口管理

### 8.1 窗口控制器

建议每个窗口一个 controller：

```text
ClipboardPanelController
PreferenceWindowController
OnboardingWindowController
PreviewPanelController
UpdateWindowController
```

控制器负责：

- 创建/销毁窗口。
- 恢复/保存 frame。
- show/hide/toggle。
- pin / auto-hide suspend。
- 生命周期事件。
- AppKit responder 链。

### 8.2 生命周期

保留当前 phase 语义，但改为 Go controller 状态：

```text
notCreated
created
ready
visible
hiddenWarm
dormant
destroyPending
destroyed
```

主窗口可 permanent，其它窗口可 idle destroy。Go 端可用 timer + generation 防止旧 timer 误销毁新窗口。

### 8.3 定位

需要复刻：

- follow cursor。
- center。
- remember。
- 多显示器。
- 全屏 Space。
- 屏幕边缘夹取。
- 预览 placement。

nsgo `OpenPanel` 没有位置参数，`Show`/`Focus` 只能使用默认的鼠标屏幕中上部定位，托盘锚定只在 `TrayOptions.Panel` 内部可用。因此定位不能依赖 nsgo 公开 API：

- 优先方案是 EcoPaste 自建 `NSPanel`，由 Go 直接控制 frame 和 collection behavior。
- 次选是使用 nsgo Panel 外壳，在 Native 根中取得 `NSWindow` 后直接 `setFrameOrigin:`/`setFrame:`；这条路径依赖 nsgo 内部句柄，必须锁定版本并隔离。

## 9. 快捷键与托盘

### 9.1 快捷键

可优先使用 nsgo `RegisterHotKey`：

- Carbon 注册，不要求辅助功能。
- 当前公开 API 没有冲突返回值，也没有暂停/恢复；需要自建 Carbon 封装或修改 nsgo。
- 冲突要反馈到 UI。
- `Alt+C` / `Alt+X` 字符串映射需要稳定。

如果 nsgo 的 key 语法/冲突处理不足，可以在平台层直接封装 Carbon。EcoPaste 的 `hotkey` 模块应把“注册结果、冲突、暂停/恢复”作为明确接口，不要让 UI 直接依赖 nsgo API。

### 9.2 托盘

使用 nsgo tray 或 AppKit `NSStatusItem`：

- template image。
- 左键菜单。
- 动态重建菜单。
- 语言切换。
- 监听暂停状态。

## 10. 备份与更新

### 10.1 备份

`internal/backup` 完全独立于 UI：

- ZIP 读写。
- Argon2id + XChaCha20-Poly1305。
- manifest。
- merge/overwrite。
- legacy import。

UI 只负责选择文件、输入密码、展示进度和结果。

### 10.2 更新

更新是迁移中最需要单独决策的模块：

| 方案 | 说明 |
| --- | --- |
| Sparkle | nsgo 原生路径；需要新 appcast 和 EdDSA key |
| 保留 Tauri updater 协议 | 需要 Go 实现兼容下载/签名校验/安装 |
| 过渡双更新器 | 旧版本继续 Tauri，新版本 Sparkle；发布流程复杂 |

不建议在未决定更新方案前重写窗口和发布链路。

## 11. 构建与分发

### 11.1 Bundle identity

必须保持：

```text
com.ayangweb.eco-paste
```

原因：

- macOS TCC 权限与 bundle id/签名相关。
- `app_local_data_dir` 与 identifier 相关。
- 旧数据路径发现依赖稳定标识。

### 11.2 构建

建议：

- `nsgo.yaml` 声明 app 名称、图标、版本、file_types、更新配置。
- Go build tags 隔离 macOS。
- 不在第一阶段添加 Windows/Linux 目标。
- 产出一个 `.app` bundle，完成 codesign、notarization、DMG。

### 11.3 文件关联

保留 `.ecopastebak` 的 UTI/MIME/file type 声明：

- macOS Info.plist `CFBundleDocumentTypes` / `UTExportedTypeDeclarations`。
- nsgo `file_types` 支持 `name`、`extensions`、`id`、`icon`、`role`、`conforms`，没有 `mimeType`；macOS 关联以 UTI 为准，Windows MIME 兼容不在本阶段范围。
- `.ecopastebak` 必须显式写 `id: com.ayangweb.eco-paste.backup`，不要依赖默认的 `<app id>.<ext>`。
- `OnOpenFiles` 接收路径。

## 12. 当前模块到目标模块的映射

| 当前 Rust/React | 目标 Go/AppKit |
| --- | --- |
| `lib.rs` | `cmd/ecopaste` + `internal/app` |
| `commands/*` | Go service/controller 方法 |
| `clipboard/read.rs` | `internal/macos/platform/pasteboard_reader` |
| `clipboard/ingest.rs` | `internal/clipboard/normalizer` + `domain` |
| `clipboard/watcher.rs` | `internal/clipboard/watcher` + app event bus |
| `clipboard/write.rs` | `internal/macos/platform/pasteboard_writer` |
| `clipboard/guard.rs` | `internal/clipboard/writeback_guard` |
| `clipboard/storage.rs` | `internal/storage/resources` |
| `clipboard/source.rs` | `internal/macos/platform/frontmost_app` |
| `clipboard/secrets.rs` | `internal/clipboard/secrets` |
| `db/*` | `internal/storage` |
| `settings/*` | `internal/settings` |
| `backup/*` | `internal/backup` |
| `window/macos.rs` | `internal/ui/clipboard_panel` + `internal/macos/window` |
| `window/lifecycle/*` | `internal/ui/window_lifecycle` |
| `shortcut/*` | `internal/macos/platform/hotkey` 或 nsgo |
| `tray/*` | `internal/ui/tray` 或 nsgo tray |
| `menu/*` | `internal/ui/menus` |
| `drag_out/*` | `internal/macos/platform/drag_out` |
| `keystroke/*` | `internal/macos/platform/paste` |
| `update.rs` | `internal/update` |
| `src/pages/*` | `internal/ui/*` |
| `src/commands/*` | `internal/app` service API |
| `src/constants/events.ts` | Go event bus 常量 |
| `src/locales/*` | Go/AppKit 资源 + 文案表 |

## 13. 目标架构的验收原则

1. 任意用户可见交互控件由 AppKit 提供，不经过 WebView。
2. SQLite 查询结果与旧版本在相同数据上语义一致。
3. 旧图片、图标和备份路径仍能解析。
4. 主窗口在不激活 EcoPaste 的情况下能接收键盘并粘贴。
5. 复制/粘贴不会产生自身回环记录。
6. 敏感内容在 Rust 旧版和 Go 新版中的处理语义一致。
7. 迁移失败不会破坏原数据库。
8. 所有平台行为在真实 macOS 桌面会话、Retina、多显示器和权限状态下验证。

## 14. 依赖与代码组织原则

### 14.1 第三方库选择

实现阶段应优先使用成熟方案，但每个依赖都要先通过以下检查：

- 维护状态：近期有提交/发布，issue 响应正常，没有长期未修复的严重问题。
- 许可证：与项目 Apache-2.0 兼容，且允许闭源分发和再分发。
- 平台兼容：支持 macOS 11+、arm64/x86_64，且不与 nsgo 的 `CGO_ENABLED=0` 默认构建冲突。
- 功能完整：能覆盖 EcoPaste 的真实边界，而不是只覆盖 demo。
- 可替换性：通过内部接口隔离，避免业务层直接依赖第三方类型。
- 构建成本：不显著增加签名、notarization、DMG、CI 和交叉编译复杂度。

候选与优先顺序：

| 领域 | 首选方向 | 说明 |
| --- | --- | --- |
| SQLite | 纯 Go 驱动 | `modernc.org/sqlite`、`ncruces/go-sqlite3` 先做真实旧库 spike；cgo 只作对照 |
| ZIP | Go 标准库 `archive/zip` | 足够覆盖当前 deflate ZIP，不需要额外依赖 |
| Argon2id | `golang.org/x/crypto/argon2` | 维护稳定，参数需与 Rust 样本逐字节对齐 |
| XChaCha20-Poly1305 | `golang.org/x/crypto/chacha20poly1305` | 与 Rust `chacha20poly1305` 容器格式对齐 |
| BLAKE3 | 维护活跃的 Go 实现 | 必须验证与 Rust `blake3` 输出一致，再决定具体库 |
| 日志 | 标准库 `log/slog` | 先满足结构化日志，确有需求再引入第三方 |
| 测试 | 标准库 `testing` + 真实样本 | 不为了测试引入复杂框架 |

不引入的依赖：

- 不引入 WebView、Electron、Wails 或任何前端运行时。
- 不把 nsgo `base` / `ui` 作为产品控件依赖。
- 不为简单工具函数引入大型框架。

### 14.2 文件与模块规模

- 一个文件只承担一个明确职责，例如“SQLite item 查询”和“备份容器加密”不要混在一起。
- 单文件接近 400–500 行时应评估拆分；超过 800 行原则上必须拆分，除非是生成代码或不可分割的协议表。
- 拆分按职责和变化原因，不按行数机械切分；不为减少行数制造大量微小文件。
- 平台代码按 `macos` build tag 隔离，业务层只依赖接口。
- AppKit 包装层、SQLite 仓储、备份格式、剪贴板领域规则分别可独立测试。
- 跨层数据用明确的 struct/接口，不传 `map[string]any` 或裸 SQL 字符串。
