# nsgo 与 AppKit 原生控件调研

## 1. 调研对象

调研基于锁定到具体 commit 的 pseudo-version：

```text
github.com/nsgo-dev/nsgo@v0.0.0-20261112101601-739024de687d
```

使用的源码入口：

- `internal/platform/macos/media.go`
- `examples/gallery/components_apple.go`
- `examples/gallery/components_macos.go`
- `cmd/nsgo/build.go`

这是 pseudo-version，不是稳定 release。README 当前明确写明：

> NSGo is early, with no release or stable API yet.

因此所有集成必须锁定具体 commit，并在 EcoPaste 内部建立隔离层，不能假设公开 API 稳定。

## 2. nsgo 的默认 macOS 渲染方式

### 2.1 不是完整的 AppKit 控件库

nsgo 默认 macOS 后端的实现说明：

> one flipped, layer-backed NSView per node, styled through its CALayer, with text the engine laid out with CoreText drawn on the main thread

含义：

- 每个节点对应一个翻转的 `NSView`。
- 背景、边框、圆角等通过 `CALayer`。
- 文本由 nsgo 用 CoreText 排版后在主线程绘制。
- `base` / `ui` 的按钮、列表等主要是 nsgo 自己绘制和布局的组件，不是稳定的 `NSButton` / `NSControl` / `NSTableView` 封装。
- 源码个别位置会借用 `NSTextField` 处理输入或粘贴，但不能据此把 nsgo 默认组件整体视为 AppKit 控件体系，也不能假设每个组件都具备原生控件语义。

因此：

- 不能把 nsgo 默认组件称为“真正的 AppKit 原生控件”。
- 不能仅通过使用 nsgo 就满足“UI 必须使用 AppKit 原生控件”的目标。

### 2.2 nsgo 仍然以 AppKit 为宿主

默认渲染仍依赖 AppKit：

- 窗口/Panel 是 `NSWindow` / `NSPanel`。
- 根 surface 是 `NSView`。
- 事件、鼠标、键盘、拖入、窗口生命周期由 AppKit 事件循环驱动。
- macOS 后端通过 `purego/objc` 调 AppKit，不需要 cgo。

所以 nsgo 适合作为应用壳和窗口宿主，但不适合作为 EcoPaste 的控件库。

## 3. 真正的原生控件入口：`core.Native`

### 3.1 公开 API

`nsgo.Component`：

```go
func Component(name string, factory core.NativeFactory) Option
```

`core.Native`：

```go
type Native struct {
    Component string
    Props     map[string]any
    PropsFunc func() map[string]any
    Style     Style
    StyleFunc func() Style
    OnEvent   func(name string, data any)
    OnPress   func(PressEvent)
    OnLayout  func(Bounds)
    Ref       *Ref
    Label     string
    TestID    string
}
```

`core.NativeFactory`：

```go
type NativeFactory func(cx *NativeContext) NativeComponent
```

`core.NativeComponent`：

```go
type NativeComponent interface {
    View() any
    SetProp(name string, value any)
    Command(name string, args any)
    Close()
}
```

### 3.2 macOS 宿主行为

`internal/platform/macos/media.go` 中的 native 配置流程：

1. 查注册的 `NativeFactory`。
2. 调 factory 创建 `NativeComponent`。
3. 取 `View()`。
4. 把返回值识别为 `objc.ID` 或 `uintptr`。
5. 设置 `autoresizingMask`。
6. 将 native view 作为 subview 加入 nsgo 节点的 `NSView`。
7. 后续 prop 变化调 `SetProp`。
8. 元素销毁时调 `Close`。

这提供了完整 AppKit 控件树的承载点。

### 3.3 官方示例已经使用 NSSlider

nsgo gallery 的 macOS 示例：

```text
examples/gallery/components_apple.go
examples/gallery/components_macos.go
```

其中通过 `core.Native` 注册并创建了真正的 `NSSlider`。这证明 `core.Native` 不是理论接口，而是现有可用通道。

### 3.4 结论

nsgo **可以直接承载完整的 AppKit 原生控件树**，入口是 `core.Native`，而不是 `base` / `ui`：

- `NativeComponent.View()` 可以返回 `objc.ID` / `uintptr`，宿主会把它加入当前窗口的 `NSView`。
- 在返回的 AppKit 视图中，可以继续创建 `NSButton`、`NSTextField`、`NSScrollView`、`NSTableView`、`NSMenu` 等原生控件。
- 这意味着不需要为了“能不能嵌入 AppKit”额外引入 Objective-C 中间层；如果 purego 的 selector、delegate、内存管理成本可控，纯 Go 方案即可。
- nsgo 本身仍只承担应用壳和原生视图承载，不提供 EcoPaste 需要的完整 AppKit 控件、表格、菜单和窗口控制器，这些必须由项目自建。

## 4. 推荐 UI 形态

### 4.1 每个窗口一个 Native 根

```text
nsgo Window/Panel
  └─ nsgo View (full-size layout host)
       └─ core.Native{Component: "ecopaste-appkit-root"}
            └─ AppKit NSView hierarchy
                 ├─ NSSplitView / NSSplitViewController
                 ├─ NSScrollView + NSTableView
                 ├─ NSSearchField
                 ├─ NSButton / NSSegmentedControl
                 ├─ NSImageView
                 └─ NSMenu / NSPopover / NSAlert
```

原则：

- 不在业务 UI 中使用 `base` / `ui` 组件。
- nsgo 只提供窗口、面板、托盘、快捷键和应用生命周期。
- 每个窗口只放一个全尺寸 `Native` 根。
- 根内部由 EcoPaste 自己的 AppKit 组件系统构建。
- nsgo 的布局只负责给根分配窗口大小；所有子控件由 AppKit Auto Layout 或 frame layout 管理。

### 4.2 为什么不把每个控件拆成一个 `Native`

如果把每个 label/button/table 都拆成独立 `core.Native`：

- nsgo 会为每个节点建立自己的 `NSView` 包装。
- AppKit 原生视图之间夹入 nsgo 的中间层，布局、first responder、menu validation、accessibility 层级都会变复杂。
- 表格 cell 复用和拖拽会话需要跨 nsgo 节点协调。
- 性能和行为更难预测。

一个窗口一个 Native 根更接近纯 AppKit 应用，也更容易保证 responder chain 和窗口语义。

### 4.3 仍需验证的边界

- `core.Native` 根是否能在窗口 resize 时无延迟填满。
- nsgo 是否在主线程调用 factory、`SetProp`、`Close`。
- 原生 view 的 `intrinsicContentSize` 与 nsgo 布局如何交互。
- nsgo surface 的根 `NSView` 是否有翻转坐标系；AppKit 子视图需要使用匹配的坐标约定。
- 输入法、Tab 键、Escape、Cmd+快捷键是否由 AppKit responder chain 正确接管。

## 5. AppKit 桥接方案

### 5.1 方案 A：纯 Go + purego/objc（推荐先验证）

做法：

- 用 `github.com/ebitengine/purego/objc` 注册类、selector、方法。
- 用 `objc.Send` 调用 `NSTableView`、`NSTextField`、`NSButton`、`NSMenu`、`NSScrollView` 等。
- 用 `objc.RegisterClass` 实现 delegate/data source。
- 用 `native` 指针在 Go map 中管理视图/控制器生命周期。

优点：

- 与 nsgo 的 `CGO_ENABLED=0` 架构一致。
- 构建、签名、交叉编译相对简单。
- 可以直接复用 nsgo 的 `purego/objc` 模式。
- 便于把 AppKit 包装层限制在 macOS build tag 内。

风险：

- selector 名称、参数编码、struct ABI 必须精确。
- delegate 生命周期、弱引用和 autorelease pool 容易出错。
- `NSTableView` 的 data source/delegate 方法多，类型签名复杂。
- 拖拽、菜单验证、输入法、KVO、通知需要大量桥接代码。
- 需要自己建立安全封装，避免业务层散落 `objc.Send`。

建议：

- 第一个 spike 只实现一个 `NSSlider` 或 `NSButton`，确认事件、prop、窗口 resize 和销毁。
- 第二个 spike 实现 `NSScrollView + NSTableView` 的最小列表，确认 delegate/data source 和 cell 复用。
- 如果两个 spike 稳定，再扩展为正式控件层。

### 5.2 方案 B：cgo + 薄 Objective-C bridge

做法：

- 写少量 `.m` / `.h` 桥接层。
- Go 通过 cgo 调用桥接函数。
- Objective-C 侧实现 delegate、target/action、拖拽和复杂控件。

优点：

- AppKit 方法签名和内存管理更自然。
- 复杂控件和 delegate 的代码量明显低于 purego。
- 更容易处理 `NSTableView`、`NSOutlineView`、`NSTextView`、拖拽和菜单验证。

风险：

- 引入 cgo，和 nsgo 的默认无 cgo 构建不同。`cmd/nsgo/build.go` 的 macOS 构建路径使用 `CGO_ENABLED=0`，采用 cgo 方案必须修改或绕开该路径，并重新验证签名、打包和 CI。
- 需要处理 build tags、C 头文件、框架链接、符号和统一日志。
- 交叉编译和 CI 复杂度上升。
- 需要明确 Objective-C 与 Go 之间的对象所有权和回调线程。

建议：

- 作为方案 A 遇到不可维护的 delegate/ABI 复杂度时的兜底。
- 把 cgo 隔离在 `internal/macos/appkitbridge`，业务层仍只依赖 Go 接口。
- 不因为引入 cgo 就把整个应用改成 Objective-C。

### 5.3 方案 C：独立 Swift/Objective-C AppKit UI + Go 核心

做法：

- AppKit UI 由 Swift/Objective-C 应用实现。
- Go 核心通过 XPC、Unix socket、stdio JSON-RPC 或嵌入动态库通信。

优点：

- AppKit 能力最完整，UI 开发最直接。
- UI 层可以使用 Xcode、Interface Builder、Swift 调试工具。

缺点：

- 不再是单一 Go 应用。
- 进程间协议、生命周期、错误传播、打包和签名都更复杂。
- 与“Go + nsgo”目标偏离较大。

建议：

- 不作为首选。
- 只有在 AppKit 桥接成本被证明不可接受时才重新评估。

### 5.4 推荐决策

```text
第一阶段：purego/objc
  -> 完成 Native 根 + NSTableView + 搜索框 + 菜单最小闭环
  -> 评估 delegate/内存/输入法成本

如果 purego 可控：
  -> 继续纯 Go AppKit 包装层

如果 purego 的 delegate/ABI 成本过高：
  -> 只把复杂控件下沉到 cgo Objective-C bridge
  -> 保留 Go 业务和 nsgo 外壳
```

## 6. nsgo 可直接提供的能力

| 能力 | nsgo API / 实现 | 可用性判断 |
| --- | --- | --- |
| 应用窗口 | `desktop.OpenWindow` | 可作外壳 |
| 无边框 Panel | `desktop.OpenPanel` | 可作剪贴板面板候选 |
| 菜单栏图标 | `desktop.NewTray` / `NSStatusItem` | 可替代 Tauri tray |
| 全局快捷键 | `desktop.RegisterHotKey` / Carbon | 可替代 global-shortcut |
| 开机自启 | `desktop.SetLaunchAtLogin` | 可用，但需 bundle |
| Dock 显隐 | `desktop.SetDockVisible` / `nsgo.HideDock` | 可用 |
| Dock reopen | `desktop.OnReopen` | 可用 |
| 文件打开 | `desktop.OnOpenFiles` + `nsgo.yaml file_types` | 可用于 `.ecopastebak` |
| 应用重启 | `desktop.Restart` | 可用 |
| 应用菜单 | `desktop` menu API | 可用 |
| 文件拖入 | `View.OnFileDrop` / `OnFileDragEnter` | 可接收备份文件 |
| 基础剪贴板文本 | `SetClipboard` / `ReadClipboard` | 不够完整 |
| 基础剪贴板图片 | `SetClipboardImage` / `ReadClipboardImage` | 不够完整 |
| 剪贴板文件读取 | `ReadClipboardFiles` | 只读，不含写回/事件 |

## 7. nsgo 当前缺口

以下能力在 EcoPaste 中必须自建，不能依赖 nsgo 默认 API：

### 7.1 剪贴板

- 剪贴板变更监听（`NSPasteboard.changeCount` 轮询或原生通知）。
- HTML/RTF 读取和写回。
- 多格式同时写回。
- 文件列表写回。
- 图片 PNG 原生字节优先读取。
- 来源应用识别。
- 敏感内容检测。
- 写回回环抑制。
- 读取失败重试。

### 7.2 系统交互

- 模拟 ⌘V。
- Accessibility / Input Monitoring 权限检测与引导。
- 拖出到外部应用。
- 原生表格、搜索框、菜单、预览、设置表单。
- 多窗口生命周期和持久化。
- 剪贴板主窗口的精确定位、pin、自动隐藏暂停。

### 7.3 数据与产品

- SQLite 仓储与 FTS5 查询。
- 备份容器和导入。
- 旧版数据导入。
- 图片/图标资源管理。
- 更新策略与 appcast 迁移。
- 多语言文案、设置迁移、使用统计。

## 8. nsgo 的 NSPanel 细节

`desktop.OpenPanel` 的公开参数只有：

```go
type PanelOptions struct {
    Width, Height Points
    HideOnBlur   bool
    Material     WindowMaterial
}
```

没有位置、屏幕、锚点、level 或 collection behavior 参数。macOS 实现创建的是：

- `NSPanel` 子类 `NsgoPanel`，`canBecomeKeyWindow` 返回 true，未覆盖 `canBecomeMainWindow`。
- style mask：`Titled | NonactivatingPanel | FullSizeContentView`。
- 透明标题栏、隐藏标题、隐藏三个 traffic lights、`setMovable(false)`、`setFloatingPanel(true)`、`setHidesOnDeactivate(false)`。
- `level = 25`（status level），collection behavior 为 `canJoinAllSpaces | fullScreenAuxiliary`。
- 内容层圆角 12、`masksToBounds = true`。
- 创建时默认隐藏；显示时 `makeKeyAndOrderFront:`。

定位行为分两种：

- `TrayOptions.Panel` 的主键点击走内部 `showAt(trayID)`：显示在托盘图标下方 6pt，并按屏幕 `visibleFrame` 夹取。这个锚点没有公开 API。
- `Window.Show()` / `Window.Focus()` 走 `showPanel(0)`：没有托盘锚点时，显示在“鼠标所在屏幕”的上中部，水平居中，垂直位于屏幕高度 2/3 再减半窗口高度，并夹到 `visibleFrame` 内。

因此：

- nsgo 可以让 panel 成为 key 而不激活 App，`HideOnBlur` 也已实现。
- 但 EcoPaste 需要的跟随光标、记忆位置、托盘锚定、指定屏幕和自定义 frame 都不能通过公开 API 表达。
- 可选路径只有：自建 `NSPanel`、或在 Native 根内通过 purego 拿到 `NSWindow` 后直接 `setFrameOrigin:`。后者依赖 nsgo 内部句柄，升级时容易失效，必须封装并做版本锁定。

但它不能直接替代 EcoPaste 的全部窗口行为：

- EcoPaste 需要跟随光标、记住位置、居中三种定位。
- 需要 pin 后失焦不隐藏。
- 需要临时暂停自动隐藏。
- 需要显示后 16ms 布局稳定。
- 需要 dormant / DestroyWhenIdle 生命周期。
- 需要预览窗口与主窗口的联动。

因此建议：

- 用 nsgo Panel 作为窗口壳。
- 用 EcoPaste 自己的 WindowController/Positioner 管理 frame、pin、hide 和生命周期。
- 必要时在 Native 根内部直接操作 `NSWindow` / `NSPanel`。

### 8.1 全局快捷键

`desktop.RegisterHotKey(key, fn)` 在 macOS 上的实际实现：

- 使用 Carbon `RegisterEventHotKey`，不要求 Accessibility 权限。
- key 先经 `driver.NormalizeKey` 归一化：小写、去掉空格、支持 `mod`/`meta`/`cmd`/`command`/`ctrl`/`control`/`alt`/`option`/`shift` 别名。
- 只支持 ANSI 键位表里的按键；未知按键会被静默跳过。
- 每次快捷键列表变化时，先注销全部，再重新注册全部。
- 只检查 `RegisterEventHotKey` 返回 0；注册失败（例如系统或其他 App 已占用）不会回调 UI，也没有 per-key 冲突结果。
- 没有公开的暂停/恢复 API。EcoPaste 录制快捷键时需要临时注销全部，这个能力必须自建 Carbon 包装或修改 nsgo。
- Carbon 热键没有 Tauri 的 Pressed/Released 双事件语义，长按重复行为也与 Tauri 插件不同，需要逐项验证。

结论：可以作为基础实现，但冲突反馈、暂停/恢复、完整键位映射和迁移兼容不能直接依赖当前 nsgo 公开 API。

### 8.2 文件打开与文件关联

`desktop.OnOpenFiles(fn)` 有冷启动排队语义：

- 文件事件在 handler 注册前到达时进入 `openPending`。
- 第一个 `OnOpenFiles` 注册后，通过 owner 的 `Post` 投递，避免在构建视图期间同步改状态。
- 已注册 handler 会收到后续每次打开事件。

这可以替代 EcoPaste 当前 `APP_READY` + pending slot 的一部分职责，但仍需在 Go 侧保存“备份文件已收到、窗口尚未就绪”的状态。

`nsgo.yaml` 的 `file_types` 字段：

```text
name
extensions
id          # UTI，默认 <app id>.<第一个扩展名>
icon        # .icns 或方形 .png
role        # editor / viewer
conforms    # 默认 public.data
```

macOS 构建会生成 `CFBundleDocumentTypes`、`UTExportedTypeDeclarations` 和 `UTImportedTypeDeclarations`。没有 `mimeType` 字段；macOS 文件关联主要依赖 UTI 和扩展名，但 Windows 关联若要保留 MIME，需要额外处理（本阶段不迁移 Windows）。

为兼容旧版本，`.ecopastebak` 必须显式设置：

```text
extensions: [ecopastebak]
id: com.ayangweb.eco-paste.backup
conforms: [public.data]
role: editor
```

### 8.3 Native 根的布局与 responder 边界

`internal/platform/macos/media.go` 的 `configureNative` 做了四件事：

1. 调用 `NativeFactory`，取得 `NativeComponent`。
2. 把 `View()` 的 `objc.ID` / `uintptr` 转成原生视图。
3. 设置 `autoresizingMask = widthSizable | heightSizable`。
4. 把原生视图 frame 设为 Native 节点的 `NSView` bounds，并 `addSubview`。

这意味着根视图能随 nsgo 分配的节点尺寸变化，但仍有边界：

- 只保证宽高跟随，不保证坐标系翻转、`intrinsicContentSize` 或 Auto Layout 约束与 nsgo 布局一致。
- Native 节点的包装 `NSView` 必须本身是全尺寸；否则根视图只会填满包装层。
- nsgo 没有替原生子树管理 first responder、Tab 键、输入法或菜单验证；窗口只设置了 `setAutorecalculatesKeyViewLoop: true`，原生控件是否能正确加入 key loop 取决于其 `acceptsFirstResponder` 和实际焦点链。
- `directTree` 会接管原生子树的 hit-test/drag 路由，但 nsgo 节点级事件不保证能收到内部 AppKit 控件事件，交互应由 AppKit target/action 或 delegate 直接处理。
- factory、`SetProp`、`Close` 的线程语义要在 spike 中确认，不能假设可以从任意 goroutine 调用 AppKit。

## 9. 更新与 Sparkle

nsgo 的更新路径：

- macOS 使用 Sparkle 2。
- `nsgo build` 在配置了 `updates` 且安装包已签名时生成 `appcast-<target>.xml`，例如 `appcast-macos_arm64.xml`。
- appcast 包含 EdDSA `sparkle:edSignature`。
- `Info.plist` 写 `SUFeedURL`、`SUPublicEDKey` 和 `SUEnableAutomaticChecks`。
- 更新由 Sparkle 自己的 UI/下载/安装流程处理。

EcoPaste 当前：

- Tauri updater endpoint。
- Tauri 自己的签名公钥和更新包格式。
- 自研更新窗口、下载进度、跳过版本。

结论：

- 不能直接复用 endpoint 或签名。
- Tauri endpoint 返回的是 Tauri updater 自定义协议，不是 Sparkle RSS appcast；`createUpdaterArtifacts` 生成的包也不能直接作为 Sparkle 安装包。
- 自动检查周期、跳过版本、下载进度窗口都属于当前应用自建逻辑，切到 Sparkle 后需要重新设计或保留自有更新器。
- 需要决定：
  1. 迁移期保留 Tauri updater 作为独立更新器。
  2. 改为 Sparkle appcast 并做发布流水线迁移。
  3. 先用独立 Go updater 保持现有服务协议，再逐步切换。

## 10. 建议的最小技术 Spike

在正式架构定稿前，只做以下 spike，不进入产品重构：

### Spike 1：nsgo + Native

- 一个窗口。
- 一个 `core.Native` 根。
- 根内一个 `NSTextField`、一个 `NSButton`、一个 `NSSlider`。
- 验证 prop、事件、resize、销毁、主线程。

### Spike 2：NSTableView

- `NSScrollView + NSTableView`。
- 1000 行数据，单列文本和图标。
- delegate/data source、cell 复用、选中、双击、右键菜单。

### Spike 3：NSPanel

- nsgo `OpenPanel` 或自建 `NSPanel`。
- show without activating app、can become key、hide on blur、pin。
- 跟随光标/居中/记忆位置、指定屏幕、托盘锚定。
- 验证 `Show`/`Focus` 的默认定位能否覆盖，以及直接改 nsgo window frame 的稳定性。

### Spike 4：剪贴板

- 读取 plain/html/rtf/image/files。
- 写回 plain/html/rtf/image/files。
- 监听 changeCount。
- 取得前台应用 bundle id。
- 模拟 ⌘V。

### Spike 5：数据

- 用候选 Go SQLite 驱动打开真实旧库。
- 执行 FTS5 trigram、LIKE 降级、WAL、trigger 更新测试。

### Spike 6：系统集成

- `RegisterHotKey` 注册成功/冲突/注销/重新注册，验证 `Alt+C`、`Alt+X` 等现有默认键。
- 录制快捷键时的暂停/恢复方案。
- `OnOpenFiles` 冷启动、运行中打开、多个 handler 的行为。
- `file_types` 生成的 Info.plist 与旧 UTI 是否一致。

只有这五个 spike 都通过，才能进入模块迁移。
