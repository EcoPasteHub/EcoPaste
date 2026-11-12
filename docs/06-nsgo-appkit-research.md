# nsgo 与 AppKit 原生控件调研

## 1. 调研对象

本机 module cache：

```text
/Users/ayang/go/pkg/mod/github.com/nsgo-dev/nsgo@v0.0.0-20261112101601-739024de687d
```

这是 pseudo-version，不是稳定 release。README 当前明确写明：

> NSGo is early, with no release or stable API yet.

因此所有集成必须锁定具体 commit，并在 EcoPaste 内部建立隔离层，不能假设公开 API 稳定。

## 2. nsgo 的默认 macOS 渲染方式

### 2.1 不是 AppKit 控件库

nsgo 默认 macOS 后端的实现说明：

> one flipped, layer-backed NSView per node, styled through its CALayer, with text the engine laid out with CoreText drawn on the main thread

含义：

- 每个节点对应一个翻转的 `NSView`。
- 背景、边框、圆角等通过 `CALayer`。
- 文本由 nsgo 用 CoreText 排版后在主线程绘制。
- `base` / `ui` 的按钮、输入框、列表是 nsgo 自己绘制和布局的组件，不是 `NSButton`、`NSTextField`、`NSTableView`。

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

- 引入 cgo，和 nsgo 的默认无 cgo 构建不同。
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

`desktop.OpenPanel` 是可行的外壳：

- 创建隐藏的无边框 `NSPanel`。
- `canBecomeKeyWindow` 返回 true。
- nonactivating style。
- status level。
- 加入所有 Space + full screen auxiliary。
- 可 `HideOnBlur`。
- 默认在托盘图标下或鼠标附近显示。
- 让 panel 成为 key 但不激活 App。

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

## 9. 更新与 Sparkle

nsgo 的更新路径：

- macOS 使用 Sparkle 2。
- `nsgo build` 生成 `appcast-macos_<arch>.xml`。
- appcast 包含 EdDSA `sparkle:edSignature`。
- `Info.plist` 写 `SUFeedURL` 和公钥。
- 更新由 Sparkle 自己的 UI/下载/安装流程处理。

EcoPaste 当前：

- Tauri updater endpoint。
- Tauri 自己的签名公钥和更新包格式。
- 自研更新窗口、下载进度、跳过版本。

结论：

- 不能直接复用 endpoint 或签名。
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
- 跟随光标/居中定位。

### Spike 4：剪贴板

- 读取 plain/html/rtf/image/files。
- 写回 plain/html/rtf/image/files。
- 监听 changeCount。
- 取得前台应用 bundle id。
- 模拟 ⌘V。

### Spike 5：数据

- 用候选 Go SQLite 驱动打开真实旧库。
- 执行 FTS5 trigram、LIKE 降级、WAL、trigger 更新测试。

只有这五个 spike 都通过，才能进入模块迁移。

