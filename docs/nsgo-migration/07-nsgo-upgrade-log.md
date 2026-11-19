# 07 · NSGo 与依赖升级记录（v0.2.0）

> 本文件记录一次完整的依赖升级：把 NSGo 从预览伪版本升级到首个稳定的 v0.2.0，并同步更新其余生产/开发依赖；同时记录按 `AGENTS.md` §4.1 对 NSGo 新 API/新特性的评估结论。
> 依据：`go.mod`、`go.sum`、NSGo v0.2.0 的 `CHANGELOG.md` 与源码、以及本仓库实际代码。

## 1. 升级范围

| 依赖 | 升级前 | 升级后 | 说明 |
| --- | --- | --- | --- |
| `github.com/nsgo-dev/nsgo` | `v0.0.0-20261112150101-70cef6a4ed17`（预览伪版本） | **`v0.2.0`**（2026-11-19 正式稳定版） | 首个稳定系列；改用稳定 tag，不再用伪版本 |
| `github.com/ebitengine/purego` | `v0.11.2` | `v0.11.3` | NSGo v0.2.0 要求 |
| `github.com/google/renameio/v2` | `v2.0.0` | `v2.0.2` | |
| `golang.org/x/image` | `v0.45.0` | `v0.48.0` | NSGo v0.2.0 要求 |
| `github.com/klauspost/cpuid/v2`（间接） | `v2.0.12` | `v2.4.2` | |
| `modernc.org/libc`（间接） | `v1.80.1` | `v1.82.2` | |
| `modernc.org/memory`（间接） | `v1.12.1` | `v1.13.0` | |
| `go` 指令 | `1.26.0` | `1.27.3` | NSGo v0.2.0 要求 Go 1.27.3+ |

其余直接依赖（`go-i18n` v2.7.0、`blake3` v0.2.4、`x/crypto` v0.59.0、`x/text` v0.44.0、`lumberjack` v2.2.1、`modernc.org/sqlite` v1.62.0）经 `go list -m -u` 确认已是各自最新稳定版本，无需变更。

`nsgo-icons-kubai` 只有伪版本、没有 tag，当前 `v0.0.0-20261112151732-a504944bd2a7` 即 `@latest`，保持不变。

`go list -m -u all` 仍会列出 `goja`、`x/net`、`x/tools`、`pprof` 等模块的新版本，但它们是 NSGo 自身模块图里被裁剪的节点，`go mod why` 证实主模块并不构建这些包，因此不强行升级（升级只会增加噪声与风险，不改变产物）。

## 2. 兼容性结论

- 升级后**无需修改任何业务或桥接源码**：`go build ./...`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`nsgo check cmd/ecopaste`、`nsgo build` 全部通过。
- 已使用的 NSGo 表面（`nsgo.Run/Component`、`core.*`、`desktop.OpenPanel/OpenWindow/NewTray/RegisterHotKey` 以外的窗口/菜单/自启/更新 API）在 v0.2.0 保持源码兼容。
- 已核对 NSGo v0.2.0 对 `NsgoPanel` 的原生行为变化，均为改进、非回归：
  - 面板在首个 engine 批次后才显示（`showFirst`），避免先闪现空标题栏；本仓库 `panelHost.open() → place() → Show()` 流程不受影响。
  - 面板 surface 默认 `WindowWorkspacesAll`（每个 Space、覆盖全屏 App），与本项目“面板在任何 Space 可用”的既有预期一致。

## 3. NSGo v0.2.0 新 API / 新特性评估（按 AGENTS.md §4.1）

逐项对照 `CHANGELOG.md` 中与桌面剪贴板应用相关的条目，并以实际代码判断是否存在“更合适的官方实现可替代自研”。

| 新特性 | 相关现有实现 | 结论 | 依据 |
| --- | --- | --- | --- |
| `desktop.FrontmostApp` / `CanReadFrontmostApp`（追踪最前台 App） | `internal/native/macos/pasteboard.go` 的 `FrontmostApp()`（同步读 `NSWorkspace.frontmostApplication`），被 `internal/app/pasteboard.go` 在采集时调用 | **不采用，保留窄桥接** | 采集在后台 goroutine 轮询到 `changeCount` 变化后立刻读取，需要“此刻”精确的前台 App；NSGo 的值是 UI 线程上的 tracked env 快照，可能滞后，且被 `core.AccessFrontmostApp` 能力门控。改用会造成来源 App 误判/需要额外授权，属于行为回归 |
| `desktop.SetWindowWorkspaces` / `WindowOptions.Workspaces` | 剪贴板面板经 `desktop.OpenPanel`（`internal/app/panel_host.go`）创建 | **已由框架默认满足，无需改动** | v0.2.0 的 `NsgoPanel` 默认 `joinsAllSpaces + fullScreenAuxiliary`，并令 panel surface `workspaces = All`。唯一自建窗口是 AppKit 子胶囊窗（`internal/native/macos/panel_nav.go`，非 NSGo surface），它按需直接设 `setCollectionBehavior:`，不属于重复实现 |
| `Window.ShowAt` / `Window.MoveTo` + `core.Placement` | `internal/native/macos/panel_placement.go` 的自定义定位（remember / center / 光标下方，多屏就近、使用整屏 frame 允许压住菜单栏/Dock，且与子胶囊窗一起移动） | **暂不采用** | NSGo 的 `ShowAt` 是“在锚点某侧摆放并夹紧到可见区域”，无法直接表达“记住任意 frame”“按屏幕居中”“光标下方”等模式，且会夹紧到可见 frame、不会同步移动自建子窗口。替换会改变既有交互 |
| `PanelOptions.FitContent` | 面板固定 420×560，内容是固定尺寸的原生视图 | **不适用** | 面板没有“随内容伸缩”的布局模型 |
| `PanelOptions.Transparent` | 面板沿用 `Material: Sidebar`（`NSVisualEffectView` 材质） | **不适用** | 透明+内容自绘圆角/阴影是另一套视觉设计，与当前毛玻璃面板冲突 |
| `ImageSource.Pixels` / `Version` | 图片预览走 `NSImageView` / `NSImage`（AppKit），不使用 NSGo `Image`/`Canvas` | **不适用** | 项目 UI 全部是原生 AppKit 控件 |
| `desktop.CanSetWindowLevel` / `CanPlaceWindows` / `SetWindowPointerEvents` | 无对应需求 | **不适用** | 仅 macOS，无需能力分支；无指针穿透需求 |

与 macOS 剪贴板应用无关、明确不评估的条目：`chat.*`、`Select`/`DatePicker` 尺寸、`SidebarShell.Fixed`、`core.ScrollEvent.Stale`、`terminal.View.HideIndicator`、`SetWebCookies`、`WebView.PinchZoom`、HarmonyOS 目标、StoreKit 内购、`base`/`ui` 组件变更等。

### 3.1 结论

v0.2.0 的桌面相关新能力主要面向 NSGo 自己的视图/面板与跨平台框架，而本项目按 `AGENTS.md` §3 的约定，UI 故意用纯 AppKit 原生控件实现（`internal/native/macos` 为叶子桥接层）。经逐项核对，本次没有“更合适且行为等价”的官方能力可以替换现有自研实现，因此**不做为用新特性而进行的重构**。若后续需求与 NSGo 能力收敛（例如面板定位模式简化、采集改到主线程），可再评估替换。

> 说明：既有自研项（Carbon 全局快捷键 `internal/native/macos/hotkey.go`、SMAppService 自启 `autostart.go`、全盘访问探测 `permission.go`）经复核 v0.2.0 仍未提供等价能力——`RegisterHotKey` 的 `State()` 是异步的、不满足设置页快捷键录入需要的同步校验；`SetLaunchAtLogin` 仍只记日志、不返回状态；NSGo 权限系统不覆盖本项目的全盘访问探测——故继续保留。

## 4. 验证清单

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 格式 | `gofmt -l .` | 通过（无输出） |
| 静态分析 | `go vet ./...` | 通过 |
| 单元测试 | `go test ./...` | 通过 |
| 并发测试 | `go test -race ./...` | 通过 |
| 模块完整性 | `go mod verify` | 全部 verified |
| NSGo 检查 | `nsgo check cmd/ecopaste` | 通过 |
| 打包 | `nsgo build -id com.ayangweb.eco-paste.dev -output build/dev cmd/ecopaste` | 通过（产出 .app 与 .dmg） |

未自动化覆盖、仍需真机手工验收：面板/托盘/全局快捷键、文本/HTML/RTF/图片/文件复制与粘贴、预览与拖出、失焦隐藏、深浅色与中英文、备份导入导出（见 `AGENTS.md` §6.2）。
