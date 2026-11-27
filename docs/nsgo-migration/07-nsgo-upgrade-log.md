# 07 · NSGo 与依赖升级记录

> 本文件记录依赖升级的依据与结论，按 `AGENTS.md` §4.1 对 NSGo 新 API/新特性做官方能力评估。
> 当前记录：**NSGo v0.2.0 → v0.3.4**（2026-11-26）。首次稳定化升级（预览伪版本 → v0.2.0）见文末「历史」小节。

## 1. 升级范围（v0.2.0 → v0.3.4）

| 依赖 | 升级前 | 升级后 | 说明 |
| --- | --- | --- | --- |
| `github.com/nsgo-dev/nsgo` | `v0.2.0` | **`v0.3.4`**（2026-11-26 正式版） | 稳定 tag |
| `github.com/nsgo-dev/nsgo-icons-kubai` | `v0.0.0-20261112151732-...`（伪版本） | **`v0.1.0`**（2026-11-26 正式版） | 首个稳定 tag；其 `go.mod` 要求 NSGo v0.3.4 / Go 1.27.3 |
| `modernc.org/sqlite` | `v1.62.0` | `v1.63.0` | |
| `modernc.org/libc`（间接） | `v1.82.2` | `v1.82.13` | |
| `modernc.org/memory`（间接） | `v1.13.0` | `v1.13.1` | |
| `golang.org/x/net`（间接） | 未直接列出 | `v0.63.0` | 随 NSGo v0.3.4 构建面进入 |
| `golang.org/x/term`、`golang.org/x/xerrors`（间接） | 未直接列出 | `v0.48.0` / `v0.0.0-20240903120638` | 随 NSGo 构建面进入 |
| `gdamore/tcell/v3`、`gdamore/encoding`、`clipperhouse/displaywidth`、`clipperhouse/uax29/v2`、`lucasb-eyer/go-colorful`、`makiuchi-d/gozxing`（间接） | 未列出 | 见 `go.mod` | NSGo v0.3.4 新增的传递依赖（终端/图标/二维码等） |
| `go` 指令 | `1.27.3` | `1.27.3` | NSGo v0.3.4 与 kubai v0.1.0 均要求 Go 1.27.3+，不变 |

其余直接依赖（`purego` v0.11.3、`renameio/v2` v2.0.2、`go-i18n` v2.7.0、`blake3` v0.2.4、`x/crypto` v0.59.0、`x/image` v0.48.0、`x/text` v0.44.0、`lumberjack` v2.2.1）经 `go list -m -u` 确认已是各自最新稳定版本，无需变更。

### 1.1 仍显示「有新版本」的模块

`go list -m -u all` 仍会列出 `untie`、`regexp2/v2`、`dop251/goja`、`pprof`、`segmentio/asm`、`sergi/go-diff`、`zeebo/assert`、`x/mod`、`x/oauth2`、`x/time`、`x/tools`、`modernc.org/gc/v3`。它们不在本模块的编译图里：`go mod why` 显示只经由依赖模块的 `*.test` 边（如 `modernc.org/sqlite.test`、`modernc.org/libc.test`、`blake3.test`）或 `go.mod` 需求出现，`go list -deps ./...` 证实主程序不构建这些包。强行升级只会增加 `go.mod` 噪声、不改变产物，故保持依赖图给出版本。

`nsgo-icons-kubai` 此前只有伪版本；v0.1.0 发布后改为稳定 tag，符合 `AGENTS.md` §4「不使用伪版本」。

### 1.2 工具链

`nsgo` CLI 必须与模块版本一致：`README.md`、`.github/workflows/ci.yml` 已从 `@v0.2.0` 改为 `@v0.3.4`。v0.3.3+ 的 CLI 在 `go.mod` 声明 `tool github.com/nsgo-dev/nsgo/cmd/nsgo` 时会自动委派给模块内版本；本项目未声明 tool（避免把 CLI 全部依赖写进 `go.mod`），因此由文档与 CI 固定安装 v0.3.4。

## 2. 兼容性结论

- **无需修改任何业务或桥接源码**：`go build ./...`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`nsgo check cmd/ecopaste`、`nsgo build` 全部通过。
- 已使用的 NSGo 表面（`nsgo.Run/Component/HideDock`、`core.*`、`desktop.OpenPanel/OpenWindow/NewTray/SetApplicationMenu/OnReopen/OnOpenFiles/Restart/SetShutdownMode`、`nsgotest`）在 v0.3.4 保持源码兼容。
- 与项目直接相关的行为变化，经核对均为改进或与现状一致：
  - `desktop.OpenPanel/OpenWindow/NewTray` 改为绑定传入的 `*core.Scope`（#1259），而不是运行中的 handler owner。本项目 `internal/app/panel_host.go` 传入应用 scope，面板不再随触发它的批次销毁，生命周期更稳定。
  - 面板帧动画只在尺寸变化且经 `desktop.SetWindowFrame` 时触发；本项目的 `applyPlacement` 只在显示后用 `setFrameOrigin:` 改位置、不改尺寸，不受影响。窄面板（宽或高 < 2×12pt）才改方角，本项目面板 420×560 保持 12pt 圆角。
  - `OnQuitRequest` 新增系统会话结束（`Forced`）语义；本项目未注册该处理器，仍走 `ShutdownOnExplicit` + `OnQuit` 清理，行为不变。
  - `desktop.Restart` 语义不变；`desktop.RequestRestart`（先询问 `OnQuitRequest`）为新增可选能力，本项目无未保存状态，不需要。
- `nsgo-icons-kubai` v0.1.0 的 `icon.go` 与升级前伪版本逐字节一致：`Symbol`/`Icon`/`Variant` API 不变，本项目现有的设置导航、空白状态、关于页图标无需改动；新增 `Icon.Click` 与更多图标资源，图标悬停动效沿用框架帧时钟。

## 3. NSGo v0.3.x 新 API / 新特性评估（按 AGENTS.md §4.1）

对照 v0.3.0–v0.3.4 的 CHANGELOG 与源码，逐项判断是否存在「更合适且行为等价」的官方实现可替换自研代码。

| 新特性 | 相关现有实现 | 结论 | 依据 |
| --- | --- | --- | --- |
| `Scope.ReadClipboardItems` / `WriteClipboardItems` / `ClipboardChangeCount` / `ClipboardFormats`（v0.3.0 #264、#2216，macOS/Windows 原生剪贴板表示） | `internal/native/macos/pasteboard.go`（AppKit `NSPasteboard` 同步读写 + `changeCount`）、`internal/clipboard/watcher.go`（后台 goroutine 每 120ms 轮询） | **暂不采用，保留窄桥接** | 新 API 经 `*core.Scope` 的引擎树异步回调：本项目的 watcher 在后台 goroutine 同步读取，并「读前读后各取一次 `changeCount`」以排除读到一半被改写；改为异步回调需要重写 watcher 并把每次轮询搬到 UI 线程。`ReadClipboardItems` 返回原始 native representation（48 MiB 配额），文本/HTML/RTF/PNG（含 TIFF→PNG）与文件 URL 仍需自行解码；写入还要把纯文本回退放进同一 item。且来源 App 追踪（`FrontmostApp`）在采集瞬间同步读取，不在该 API 内。替换会造成采集时序/来源判定回归 |
| `desktop.QuickLook` / `CanQuickLook`（#370） | `internal/native/macos/preview.go`：面板内 `NSPopover` + `NSTextView`/`NSImageView` 预览文本、富文本、图片 | **不采用** | QuickLook 是系统文件预览面板（Space/Escape 关闭），不能替换面板内的就地内容预览；若为文件项新增系统预览属新交互，超出本次依赖升级范围 |
| `desktop.Monitors` / `desktop.CursorPoint`（v0.3.0） | `internal/native/macos/panel_placement.go` 的 `screensBounds()`/`mouseLocation()`，同步在 AppKit 主线程计算面板位置，并与子胶囊窗一起移动 | **不采用** | 定位逻辑需要「显示瞬间」的同步值，且刻意使用整屏 frame（允许压住菜单栏/Dock）、AppKit 左下角全局坐标；`CursorPoint` 是异步回调，`Monitors` 的坐标空间/可见区域语义不同，替换会改变定位行为并拆分同步路径 |
| `desktop.PanelDock` / `CanDockPanels`（v0.3.4） | 面板定位模式：remember / center / 光标下方（`panelOrigin`） | **不适用** | 边缘停靠是另一种定位模式，与现有三种模式不等价；是否增加属产品决策 |
| `nsgo.StartHidden`（v0.3.0 #670） | 应用无主窗口：`nsgo.Run(app.Run, nsgo.HideDock())`，面板按需 `Show` | **不适用** | 启动时没有主窗口需要隐藏 |
| `desktop.RequestQuit` / `RequestRestart` + `QuitRequest.Forced`（v0.3.0） | `desktop.SetShutdownMode(..., ShutdownOnExplicit)`、`app.OnQuit` 清理、`desktop.Restart` | **不采用** | 本项目没有需要确认的未保存状态，也没有 `OnQuitRequest` 处理器；直接退出/重启的语义即所需 |
| `desktop.MenuIcon`（v0.3.x） | `internal/menu` 构造的文字菜单项 | **不采用** | 当前菜单设计为纯文字；加图标属视觉调整，非升级必需 |
| `desktop.StandardDir` / `UserDir`（v0.3.x） | `internal/storage` 自管理数据目录（支持自定义目录、`storage.json` 锚点） | **不适用** | 项目的数据目录契约是 2.0.0 基线的一部分，不能换成系统标准目录 |
| 新增 `PermissionContacts/Photos/Automation/Notifications`（v0.3.0） | `internal/native/macos/permission.go` 全盘访问探测 | **不适用** | NSGo 权限系统仍不覆盖本项目需要的全盘访问探测，继续保留自研 |
| `core.Style.GlassInteractive`、`ui/terminal`、`gpu`、`cloud`、`gameservices`、`gamepad` 等 | 无 | **不适用** | UI 为纯 AppKit，不使用 NSGo 视图/GPU/终端组件，当前只支持 macOS |

### 3.1 结论

v0.3.x 的新能力集中在跨平台视图形状、GPU、终端、云与游戏服务，与本项目的 AppKit 原生剪贴板应用不重合；唯一直接相关的是原生剪贴板表示 API，但其异步模型、来源 App 缺失与表示解码负担都劣于现有窄桥接。因此**本次不为用新特性而重构**，仅升级版本。

## 4. 新增构建门禁：第三方许可证声明

v0.2.0 的 `nsgo build` 不检查许可证；v0.3.4 会分类依赖许可证、生成并打包 `THIRD_PARTY_NOTICES.txt`，且默认 `licenses.unknown: error`。升级后首次打包因两个模块的附加文件报错：

- `modernc.org/memory` 的 `LICENSE-LOGO` 只有一个 Wikimedia 徽标链接（署名引用，不是许可证正文），分类器无法识别；
- `modernc.org/sqlite` 的 `LICENSE-SQLITE` 是 SQLite 公有领域声明，分类器同样无法识别；其扁平化的 `LICENSE-3RD-PARTY.md` 里还含 MPL-2.0 文本，但对应的是 `hashicorp/golang-lru`（`go list -deps ./...` 证实并未链接进程序，仅测试期依赖）。

复核后只对这两个模块做精确豁免，其余模块仍保持 `unknown: error`：

```yaml
licenses:
  allow:
    - modernc.org/memory
    - modernc.org/sqlite
```

豁免不会隐藏声明：两份文本仍完整进入 `Contents/Resources/THIRD_PARTY_NOTICES.txt`（本机打包已确认，约 4254 行）。

另外，v0.3.4 打包会在 `cmd/ecopaste/.nsgo/` 生成 `notices-*.go`（已由 `.gitignore` 忽略）。`gofmt -l .` 会遍历该目录并误报未格式化，因此 `Makefile` 的 `fmt` 目标、CI 的格式检查以及 `AGENTS.md` 的验证命令都改为只检查 Git 索引与未忽略的新增 Go 文件。

## 5. 验证清单

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 格式 | `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')` | 通过（无输出） |
| 静态分析 | `go vet ./...` | 通过 |
| 单元测试 | `go test ./...` | 通过 |
| 并发测试 | `go test -race ./...` | 通过 |
| 模块完整性 | `go mod verify` | 全部 verified |
| 模块整洁 | `go mod tidy -diff` | 无差异 |
| NSGo 检查 | `nsgo check cmd/ecopaste`（v0.3.4 CLI） | 通过（catalogs + vet） |
| NSGo 无头测试 | `nsgo test cmd/ecopaste`（v0.3.4 CLI） | 通过 |
| 打包 | `nsgo build -id com.ayangweb.eco-paste.dev -output <tmp> cmd/ecopaste`（v0.3.4 CLI） | 通过（.app + .dmg，含 THIRD_PARTY_NOTICES.txt） |

未自动化覆盖、仍需真机手工验收：面板/托盘/全局快捷键、文本/HTML/RTF/图片/文件复制与粘贴、预览与拖出、失焦隐藏、深浅色与中英文、备份导入导出（见 `AGENTS.md` §6.2）。

## 6. 历史：v0.2.0（2026-11-19）

首个稳定化升级：把 NSGo 从预览伪版本 `v0.0.0-20261112150101-70cef6a4ed17` 固定到 `v0.2.0`，同步升级 `purego` v0.11.2→v0.11.3、`renameio/v2` v2.0.0→v2.0.2、`x/image` v0.45.0→v0.48.0、`libc` v1.80.1→v1.82.2、`memory` v1.12.1→v1.13.0，`go` 指令 1.26.0→1.27.3。当时评估了 `desktop.FrontmostApp`、`SetWindowWorkspaces`、`Window.ShowAt/MoveTo`、`PanelOptions.FitContent/Transparent`、`ImageSource.Pixels` 等，结论均为保留自研桥接或框架已默认满足，未做重构。
