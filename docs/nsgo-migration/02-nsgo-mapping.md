# 02 · NSGo 能力映射与目标架构

> NSGo 源码快照：`github.com/nsgo-dev/nsgo`（`oneo-me/nsgo` 的规范模块路径）。
> 关键事实来自 `ARCHITECTURE.md`、`docs/architecture/*`、`core/`、`desktop/`、`internal/platform/macos/`。

## 1. NSGo 是什么

NSGo 是一个 **Go 原生 UI 引擎**，不是 Tauri 的前端替代品：

- 应用逻辑、响应式图、样式、布局和文本在 Go 中运行。
- 每个平台用自己的原生 UI 绘制；macOS 通过 `purego` + Objective-C runtime 直接调用 AppKit，官方要求 `CGO_ENABLED=0`。
- 入口是 `nsgo.Run` / `nsgo.SetMainView` / `nsgo.NewView`。
- 桌面能力在 `desktop` 包：窗口、面板、托盘、全局快捷键、菜单、开机自启、Dock 策略。
- UI 结构在 `core`：`View`、`Style`、`Text`、`List`、`Each`、`Show`、`Dynamic`、signals、regions。
- 原生视图通过 `core.Native` + `nsgo.Component(name, factory)` 注册；macOS 组件返回 `NSView`。
- 平台服务通过 `core.Scope`：剪贴板、打开链接、文件选择、通知、外观、生命周期等。
- 更新通过 `updater` + 平台 updater：macOS 是 Sparkle，Windows 是 WinSparkle。
- 包管理/构建通过 `nsgo.yaml` + `nsgo` 工具：`nsgo dev`、`nsgo check`、`nsgo test`、`nsgo build`。

官方状态：**早期项目，没有 release 或稳定 API**。这是迁移的第一风险。

## 2. NSGo 与现有 Tauri/React 的概念映射

| 现有 | NSGo | 说明 |
| --- | --- | --- |
| Tauri 主进程 | `nsgo.Run` 的 app scope | 一个 Go 进程、一个 engine；窗口是 engine 的 surface |
| Tauri 多 WebView 窗口 | `desktop.OpenWindow` / `desktop.OpenPanel` | 窗口共享同一个 engine 和 signals |
| Tauri NSPanel | `desktop.OpenPanel` / 自定义 `NSPanel` 组件 | NSGo 的 panel 支持 `HideOnBlur`、非激活、状态层 |
| Tauri tray | `desktop.NewTray` | macOS `NSStatusItem`；支持图标/view、panel、menu |
| Tauri global shortcut | `desktop.RegisterHotKey` | macOS Carbon `RegisterEventHotKey`，不需要辅助功能权限 |
| Tauri app menu | `desktop.SetApplicationMenu` | macOS `NSMenu` |
| Tauri context menu | `desktop.PopupMenu` | macOS `NSMenu`；动作通过 ID 回到 Go |
| Tauri autostart | `internal/native/macos` 的 `SMAppService` 状态感知桥接 | NSGo 封装忽略失败且读取 `NSError`；自研桥接避免 `NotFound` 时误 unregister |
| Tauri single instance | 数据根 `flock` + `desktop.OnReopen` | 二次实例通过 `SIGUSR1` 让主实例打开设置后退出 |
| Tauri file open | `desktop.OnOpenFiles` + `nsgo.yaml` `file_types` | macOS `application:openURLs:` |
| React 组件树 | NSGo `core` elements + signals | 组件只运行一次，signal 驱动局部 region 重建 |
| Valtio | NSGo signals / `core.Storage` | UI 状态用 signal；持久化用 JSON/SQLite |
| Ant Design 控件 | AppKit 原生控件（`NSButton`、`NSSearchField`、`NSTableView` 等） | 这是本项目 UI 规范的核心 |
| React 虚拟列表 | `NSTableView` / `NSCollectionView`（原生组件）+ `core.List` 仅做结构 | 历史列表需要原生虚拟化和键盘导航 |
| HTML/RTF 渲染 | `NSAttributedString` + `NSTextView` | 不使用 WebView |
| 图片预览 | `NSImageView` / `NSImage` | 支持 PNG/TIFF/JPEG/HEIC/GIF |
| 文件图标 | `NSWorkspace.icon(forFile:)` | 输出 PNG 缓存 |
| 拖出 | `NSDraggingSession` + `NSPasteboardItem` | 文件/图片/文本多格式 |
| 粘贴模拟 | `CGEvent` ⌘V | 需要辅助功能权限 |
| SQLite | `modernc.org/sqlite` | 纯 Go、无 CGO；已验证支持 FTS5 trigram |
| JSON 设置 | Go `encoding/json` + 原子 rename | 保持 camelCase 与 patch 语义 |
| `.ecopastebak` | Go `archive/zip` + `x/crypto/argon2` + `ChaCha20-Poly1305` | 保持容器字节格式 |
| Tauri updater | NSGo `updater` + Sparkle appcast | 现有 update JSON endpoint 不能直接复用 |
| Tauri i18n | NSGo `i18n` / Go 文案目录 | 复用现有翻译内容 |
| Vite/HMR | `nsgo dev` | Go 代码热重载；不涉及 WebView |

## 3. NSGo 已提供、可直接使用的能力

### 3.1 桌面窗口与面板

`desktop.OpenPanel` 的 macOS 实现是 `NsgoPanel`：

- `NSPanel`，可成为 key 但不激活 App。
- 透明标题栏、无控制按钮、圆角、阴影。
- 默认 status level、所有 Space 可见。
- `HideOnBlur` 失焦隐藏。
- 可放在托盘图标下方，或屏幕上方居中。

这与剪贴板窗口的需求高度接近，但还需要补充：

- 固定窗口 / 粘贴时 resign/restore key。
- 跟随鼠标定位、记住位置、多屏。
- 透明背景 + `NSVisualEffectView` 材质。
- 失焦隐藏时区分“编辑中/文件面板打开/右键菜单打开”。

### 3.2 托盘

`desktop.NewTray` 支持：

- 图标或自绘 view。
- 左键 panel / 右键 menu。
- menu 可以每次点击时生成。
- template image（macOS 菜单栏）。

现有托盘菜单项可直接映射：偏好、暂停/开始监听、开源地址、检查更新、版本、重启、退出。

### 3.3 全局快捷键

`desktop.RegisterHotKey(cx, "alt+c", fn)`：

- macOS Carbon，不需要辅助功能权限。
- 支持作用域 owner；owner 结束时注销。
- `KeyLabel` 可生成平台快捷键文案。

现有 `Alt+C` / `Alt+X` 可直接迁移；快捷键录制器仍需要自定义 `NSSearchField`/`NSTextField` 捕获键盘。

### 3.4 菜单

`desktop.SetApplicationMenu` / `desktop.PopupMenu`：

- macOS `NSMenu`。
- 角色项（About/Hide/Quit/Cut/Copy/Paste/Select All 等）由平台本地化。
- 普通项通过 ID 回到 Go。
- 子菜单、separator、accelerator 都支持。

现有右键菜单的动作分组、快捷键、分组子菜单可以原样表达。

### 3.5 剪贴板服务

`core.Scope` 自带：

- `ReadClipboard` / `SetClipboard`：纯文本。
- `ReadClipboardImage` / `SetClipboardImage`：PNG + 尺寸；上限 48MB。
- `ReadClipboardFiles`：`file:` URL 列表。
- `HasClipboardText` / `HasClipboardImage`。

**缺口**：

- 没有 HTML / RTF 读取与写入。
- 没有文件写入（只有读取）。
- 没有剪贴板变更监听。
- 没有多格式同时写入（plain + HTML + RTF）。
- 没有 `changeCount` / 回环抑制。

因此剪贴板模块必须实现一个 **AppKit 原生 `NSPasteboard` 服务**，不能只用 NSGo 内置服务。

### 3.6 原生组件桥

`core.Native`：

- `Component` 名称对应 `nsgo.Component` 注册的工厂。
- `Props` / `PropsFunc` 跨到原生组件的 `SetProp`。
- `OnEvent` 接收原生组件的 `Emit`。
- `Ref.Command` 向原生组件发命令。
- 原生组件可 `SetSize` 报告自然尺寸。
- 动画值可以绑定到原生 prop。

这足以承载一套 AppKit 组件库：

- `NSButton` / `NSSegmentedControl` / `NSSwitch` / `NSSlider` / `NSPopUpButton`
- `NSSearchField` / `NSTextField` / `NSTextView`
- `NSTableView` / `NSCollectionView`
- `NSImageView`
- `NSProgressIndicator`
- `NSBox` / `NSVisualEffectView`

### 3.7 更新

`updater` + `nsgo build`：

- macOS 使用 Sparkle 2.10.1。
- 需要一个 appcast XML + EdDSA 签名。
- `nsgo.yaml` 的 `updates.feed` / `updates.key` 配置。
- 支持 direct / store / none 渠道。

现有 Tauri updater 使用 `https://releases.ecopaste.cn/update?channel=...` JSON endpoint，不能直接给 Sparkle 使用。需要二选一：

1. 在现有发布服务增加 Sparkle appcast（推荐，复用 Sparkle 的安装/重启体验）。
2. 自己实现更新检查 + 下载 + 替换 .app（工作量大，且要处理签名/回滚）。

### 3.8 打包与签名

`nsgo build` 支持：

- macOS `.app` / `.dmg`。
- `sign.identity`、公证、Sparkle appcast。
- `resources/` 资源根。
- `nsgo.yaml` 的 `file_types` 文件关联。

现有 GitHub Actions 的 release/nightly 需要改为 Go/NSGo 构建矩阵；Windows 构建可以先保留旧流水线，macOS 新流水线独立。

## 4. 必须自建或补桥的能力

| 能力 | 原因 | 建议实现 |
| --- | --- | --- |
| `NSPasteboard` 全格式读写 | NSGo 只覆盖 text/image/files | `internal/native/macos/pasteboard`：`NSPasteboard` + `NSPasteboardItem`，支持 `public.utf8-plain-text`、`public.html`、`public.rtf`、`public.png`、`public.tiff`、`public.file-url` |
| 剪贴板变更监听 | `NSPasteboard` 无公开通知 | 后台 goroutine 轮询 `changeCount`，120ms；变化后同步抓前台应用，再 `Scope.Post` 入库 |
| HTML/RTF 写回 | 需要多表示 | 一次 `NSPasteboardItem` 写 plain + HTML/RTF；不要分次 clear |
| 文件写回 | NSGo 不支持 | `NSPasteboardItem` + `public.file-url`，或 `NSFilenamesPboardType` 兼容 |
| 图片字节哈希 | 去重必须与旧数据一致 | 读取 PNG 原始字节，`blake3("image:"+file_name)`；落盘文件名用 PNG 字节 blake3 |
| 文件图标 | NSGo 无 | `NSWorkspace.icon(forFile:)` → `NSImage` → PNG；按 cache key + platform 缓存 |
| 应用图标 | NSGo 无 | `NSWorkspace.icon(forApplication:)` 或 `NSRunningApplication.icon` |
| 拖出 | NSGo 无 | `NSDraggingSession`，文件/图片/文本；保留现有 128pt 预览图策略 |
| 粘贴模拟 | NSGo 无 | `CGEvent` ⌘V；检查 `AXIsProcessTrustedWithOptions` |
| 权限检查 | NSGo 无 macOS 权限服务 | `ApplicationServices`（辅助功能）、`NSFileManager`/TCC（完全磁盘访问）、`SMAppService`（自启） |
| 旧版导入 | 纯业务 | Go 打开 v0.6 `history` 表，复用数据映射规则 |
| 备份加密 | Go 标准库可做 | `archive/zip` + `golang.org/x/crypto/argon2` + `chacha20poly1305` |
| 更新 appcast | 现有 endpoint 不兼容 | 增加 appcast 生成；保留旧 Tauri endpoint 直到切换 |
| 日志 | NSGo 无统一日志 | 标准 `log` + `lumberjack` 轮转（5 MB × 3，14 天），路径 `<data_root>/logs`，写入前脱敏 |
| 单实例 | 数据根 `flock`；`desktop.OnReopen` 处理 LaunchServices 重入 | 直接执行第二个二进制也会被拦截，避免两个进程同时写 DB |
| 防默认行为 | NSGo 无 WebView | 不需要；键盘事件由 AppKit/NSGo 处理 |
| 轻量化 | NSGo 无 WebView 重建 | 不需要销毁 WebView；可改为释放大图缓存/暂停刷新 |

## 5. 关键差距与风险

### 5.1 NSGo 早期 API 不稳定

- 没有 release / 稳定 API。
- `core`、`desktop`、`internal/platform/macos` 都可能改。
- 迁移文档必须固定 commit，并在 `go.mod` 中用 `replace` 指向可控 fork 或固定 commit。
- 建议在 `internal/native` 与 NSGo 之间保留 adapter，不让业务层直接依赖 `core.Native` 细节。

### 5.2 AppKit 桥接是主要工作量

- NSGo 只提供 `core.Native` 桥，不提供完整 AppKit 控件库。
- 需要自己实现 `NSTableView` 数据源/委托、`NSSearchField` 事件、`NSTextView` 富文本、`NSMenu` 动态菜单、`NSDraggingSession`。
- 必须严格遵守“AppKit 只在主线程”的规则；Go 侧用 `dispatch_async` 到 main queue。

### 5.3 线程模型

- NSGo engine 在专用 goroutine；AppKit 在主线程。
- 剪贴板 watcher 在后台 goroutine。
- 不能从 watcher 直接操作 AppKit；用 `core.Scope.Post` 或 `engine.Dispatch` 回到 engine，再发 mount item 到主线程。
- SQLite 写入不能在主线程；用后台 worker + 单写连接。

### 5.4 数据兼容

- 现有 DB 是 SQLx `_sqlx_migrations` 管理；Go 侧不能假设自己的 migration 表。
- 建议首次启动做“基线检测”：检测 `clipboard_items` 与 `_sqlx_migrations`，备份数据库，再建立 Go 的 `schema_migrations`，把 0001 标记为已应用。
- 在双跑期不要改 schema；如果必须改，同时写 SQLx 兼容 migration 或推迟到 cutover。
- 时间字符串、`content_hash`、图片文件名、FTS 触发器必须逐字节兼容。

### 5.5 FTS5

- 已用 `modernc.org/sqlite` 实测：SQLite 3.54.1，FTS5 trigram 可用。
- trigram 对 <3 字符查询无效，必须保留 `LIKE` 兜底。
- 中文按字符计数，不是字节。

### 5.6 富文本

- 现有列表不渲染 HTML/RTF，只显示纯文本摘要；预览也用纯文本。
- 迁移后如果想“更原生”，可以用 `NSAttributedString` 渲染 HTML/RTF，但这会改变现有行为，应作为后续增强，不是 parity 必需。
- 写回必须保留 plain fallback，否则很多 App 粘贴失败。

### 5.7 更新

- Sparkle 需要 appcast、签名、EdDSA key，现有 Tauri updater 的 JSON endpoint 不兼容。
- 这是发布链路的硬切换，必须单独排期。

### 5.8 主题与视觉

- 现有 UI 是 Ant Design token + UnoCSS；迁移后要用 AppKit 语义色（`NSColor.labelColor`、`controlBackgroundColor`、`separatorColor`、`selectedContentBackgroundColor`）。
- 不要用 NSGo `ui` 组件模拟 AppKit 外观；外观由 AppKit 控件决定。

### 5.9 可访问性

- AppKit 原生控件自带轴信息，但自定义卡片/列表仍要设置 `accessibilityLabel`、`accessibilityRole`。
- NSGo `core` 的 accessibility 与 AppKit 原生控件需要合并，避免重复朗读。

### 5.10 打包与签名

- NSGo `nsgo build` 使用 Xcode 工具链、Sparkle、公证。
- 现有 Developer ID / notarization / release secrets 需要迁移。
- 迁移期可以继续用 Tauri 发版，NSGo 只做本地/内部构建。

## 6. 迁移策略选项

### 方案 A：Go 重写核心 + NSGo 原生 UI（推荐）

**做法**：把 Rust 业务逻辑逐模块迁到 Go；NSGo 负责应用结构、状态、布局、窗口/托盘/快捷键；AppKit 负责控件和系统交互。

**优点**：

- 单一运行时，无 IPC / 双进程。
- 最符合 NSGo 设计。
- 长期维护成本最低。
- 可以完全掌控 AppKit 原生体验。

**缺点**：

- 需要重写约 22.5k 行 Rust 的业务逻辑。
- AppKit 桥接工作量大。
- 需要逐项兼容数据/备份/更新。

**结论**：作为主路线。

### 方案 B：NSGo UI + Rust core sidecar

**做法**：保留 Rust 核心为独立进程，通过 Unix socket / localhost JSON-RPC 与 Go/NSGo 通信。

**优点**：

- 核心业务不用重写。
- 可以先验证 NSGo UI。

**缺点**：

- 剪贴板监听/写回/拖出/粘贴模拟跨进程，时序和权限复杂。
- 打包、签名、生命周期、崩溃恢复都更复杂。
- 两个运行时，长期维护成本高。
- 不符合“原生体验优先”的目标。

**结论**：只作为极端回退方案，不作为长期架构。

### 方案 C：Rust staticlib + cgo

**做法**：把 Rust 编译为静态库，Go 通过 cgo 调用。

**缺点**：

- NSGo macOS 后端要求 `CGO_ENABLED=0`，直接冲突。
- 构建链、符号、panic、线程、内存管理都复杂。

**结论**：不推荐。

### 方案 D：保留 Tauri，嵌入 NSGo

**做法**：在 Tauri WebView 中嵌入 NSGo 渲染的 NSView。

**缺点**：

- 仍然是 WebView + React，不满足“弃用 Tauri UI”的目标。
- NSGo 与 Tauri 窗口/事件循环叠加，复杂度高。

**结论**：不采用。

## 7. 目标架构

### 7.1 目录

```text
cmd/ecopaste/                 # NSGo app entry, macOS build tag
internal/app/                 # app wiring, windows, lifecycle, commands
internal/ui/                  # NSGo views: layout/state only
internal/native/macos/        # AppKit components + platform services
internal/native/macos/pasteboard/
internal/native/macos/controls/
internal/native/macos/drag/
internal/native/macos/icons/
internal/clipboard/           # watcher, payload, read, write, ingest, dedupe, guard, secrets
internal/database/            # sqlite, migrations, repositories
internal/settings/            # model, store, patch/merge, validation
internal/backup/              # .ecopastebak, zip, argon2, chacha
internal/storage/             # data dirs, image/app/file icon stores, usage, relocation
internal/keystroke/           # CGEvent paste
internal/menu/                # NSMenu construction / actions
internal/tray/                # tray model / menu
internal/shortcut/            # hotkey registration / recording
internal/update/              # Sparkle/appcast integration
internal/i18n/                # Go catalogs
internal/onboarding/          # legacy import, permissions
internal/legacy/              # optional reference, not compiled
```

### 7.2 依赖方向

```text
cmd/ecopaste
  -> internal/app
       -> internal/ui
       -> internal/native/macos
       -> internal/clipboard
       -> internal/database
       -> internal/settings
       -> internal/backup
       -> internal/storage
       -> internal/update
       -> internal/i18n

internal/ui
  -> nsgo/core
  -> internal/app services (interfaces)

internal/native/macos
  -> purego/objc
  -> AppKit / Foundation / CoreGraphics
  -> no database, no settings persistence
```

规则：

- `internal/native/macos` 是叶子层，只做系统调用和视图，不读 DB、不写设置。
- `internal/ui` 不直接访问 SQLite / 文件系统；通过 app service 接口。
- `internal/clipboard`、`internal/database`、`internal/settings` 不依赖 AppKit。
- `cmd/ecopaste` 只做 wiring。

### 7.3 数据流

**采集**：

```text
NSPasteboard changeCount poll
  -> read all available representations
  -> capture order / size / filters / secrets
  -> build payload (plain, html, rtf, image bytes, file paths)
  -> content hash
  -> SQLite upsert
  -> UI signal / list refresh
```

**列表**：

```text
NSTableView data source
  -> page query (SQLite FTS/LIKE)
  -> enrich (thumbnail, app icon, file icons, display time, actions)
  -> native row view
```

**粘贴**：

```text
user action
  -> load full item
  -> write NSPasteboard (multi-format)
  -> hide/resign panel
  -> CGEvent Cmd+V
  -> restore panel key if pinned
```

**备份**：

```text
SQLite + resources + settings
  -> zip payload
  -> optional Argon2id + ChaCha20-Poly1305
  -> .ecopastebak
```

### 7.4 UI 组件清单

建议优先实现这些 AppKit 原生组件（`internal/native/macos/controls`）：

| 组件 | AppKit | 用途 |
| --- | --- | --- |
| 搜索框 | `NSSearchField` | 历史搜索 |
| 分段/标签 | `NSSegmentedControl` | 分类 Tab |
| 下拉 | `NSPopUpButton` | 排序、分组、主题、语言 |
| 开关 | `NSSwitch` | 布尔设置 |
| 按钮 | `NSButton` | 普通/危险/图标按钮 |
| 文本输入 | `NSTextField` / `NSTextView` | 备注、快捷键、数字 |
| 列表 | `NSTableView` | 历史记录虚拟化、键盘导航 |
| 卡片 | 自定义 `NSTableCellView` / `NSView` | 文本/图片/文件卡片 |
| 图片 | `NSImageView` | 缩略图、预览 |
| 进度 | `NSProgressIndicator` | 更新下载、存储统计 |
| 弹窗 | `NSAlert` / `NSPanel` | 删除确认、密码、错误 |
| 文件选择 | `NSOpenPanel` / `NSSavePanel` | 备份、图片另存、数据目录 |
| 菜单 | `NSMenu` | 右键、托盘、应用菜单 |
| 富文本 | `NSTextView` + `NSAttributedString` | RTF/HTML 预览（可选） |

### 7.5 状态与持久化边界

- **SQLite**：历史、分组、来源应用、文件图标缓存。唯一业务数据源。
- **settings.json**：用户偏好。Go struct + JSON patch，保持 camelCase。
- **NSGo signals**：当前窗口的 UI 状态（搜索词、选中项、滚动位置、加载状态）。
- **`core.Storage`**：只放小型 UI 状态（如上次窗口选择），不要放历史。
- **图片/图标**：文件系统 + DB 引用；不要塞进 SQLite blob。

## 8. 迁移前后的兼容策略

### 8.1 保留同一身份

- Bundle ID：`com.ayangweb.eco-paste`。
- 数据目录：`<app_local_data>/<dev|prod>`，沿用 `storage.json`。
- DB 文件名：`db/clipboard.db`。
- 设置文件：`config/settings.json`。
- 资源目录：`resources/clipboard-images`、`app-icons`、`file-icons`。
- 备份扩展名与格式：`.ecopastebak`。
- 文件关联：`.ecopastebak`。

### 8.2 开发期隔离

- NSGo 开发版使用独立 bundle ID / 数据目录（例如 `com.ayangweb.eco-paste.dev` + `EcoPasteDataDev`），避免与已安装 Tauri 正式版互相污染。
- 兼容性测试使用真实数据目录的**副本**，不直接在生产数据上试迁移。

### 8.3 双跑期

- Tauri 版继续发版，NSGo 版只做内部构建。
- 两边都不改 schema；如果必须改，先写兼容 migration，再在 NSGo 中实现同样变更。
- 备份格式保持只读兼容：NSGo 能读 Tauri 备份，Tauri 也能读 NSGo 备份。
- 直到 NSGo 版通过 parity checklist，才切换默认下载。

### 8.4 切换后

- 保留一个版本的旧版导入入口，读取旧数据目录。
- 保留 `.ecopastebak` 导入导出。
- 旧 Tauri 代码归档到 `legacy/` 或单独分支，不留在主构建路径。

## 9. 结论

- NSGo 能承担“应用结构、状态、布局、窗口/托盘/快捷键/菜单”的职责。
- AppKit 必须承担“控件、列表、富文本、拖拽、粘贴、图标、权限”等原生体验。
- 现有 Rust 业务不能直接复用；推荐 Go 重写，而不是 sidecar/cgo。
- 迁移的第一优先级不是 UI，而是 **数据契约 + 剪贴板管线 + macOS 系统交互**。
