# EcoPaste Go + nsgo 重构分析

> 分析基线：`master` 上的 `5139d30b chore: update dependencies (#1410)`
>
> 分析分支：`refactor/go-nsgo-analysis`
>
> 阶段状态：第一阶段，仅完成源码分析与技术文档；未编写 Go 代码、未创建 AppKit UI、未修改现有业务逻辑。

## 1. 文档目标

这套文档用于回答四个问题：

1. EcoPaste 当前到底实现了哪些功能和业务规则。
2. Rust、Tauri、React 各模块如何协作，哪些能力必须保留，哪些属于 WebView 时代的实现细节。
3. nsgo 能否支撑“Go + 真正的 AppKit 原生控件 + 无 WebView”的 macOS 版本。
4. 从当前版本迁移时，数据库、图片资源、设置和 `.ecopastebak` 备份如何保持兼容。

文档以当前源码为唯一事实来源。`.trellis/spec/` 和既有说明仅作为背景，不作为实现依据；发现文档与源码不一致时，以源码为准，并在对应章节标注。

## 2. 文档索引

| 文档 | 内容 |
| --- | --- |
| [01-current-architecture.md](01-current-architecture.md) | 当前技术栈、启动顺序、Rust 模块职责、前后端契约与运行时模型 |
| [02-feature-inventory-and-flows.md](02-feature-inventory-and-flows.md) | 功能清单、剪贴板全链路、列表/搜索/收藏/置顶/分组、粘贴与清理流程 |
| [03-data-storage-and-compatibility.md](03-data-storage-and-compatibility.md) | 数据目录、SQLite schema、FTS5、图片资源、设置持久化与兼容要求 |
| [04-backup-restore-and-legacy-data.md](04-backup-restore-and-legacy-data.md) | `.ecopastebak` 容器、加密格式、merge/overwrite 导入、旧版历史数据导入 |
| [05-macos-platform-integration.md](05-macos-platform-integration.md) | NSPanel、窗口生命周期、快捷键、托盘、粘贴模拟、拖出、权限与更新 |
| [06-nsgo-appkit-research.md](06-nsgo-appkit-research.md) | nsgo 版本与底层实现、`core.Native`、原生控件桥接方案、现有能力缺口 |
| [07-target-go-nsgo-architecture.md](07-target-go-nsgo-architecture.md) | 目标 Go + nsgo + AppKit 架构、模块映射、数据访问、UI 分层与构建方案 |
| [08-migration-plan-and-risks.md](08-migration-plan-and-risks.md) | 推荐迁移顺序、验证门槛、风险清单与尚需决策 |

## 3. 核心结论

### 3.1 当前项目

- 当前是 **Tauri v2 + Rust-First + React 19**：Rust 负责剪贴板、数据库、窗口、快捷键、托盘、菜单、拖出、备份和更新；React 负责渲染与普通键盘交互。
- 业务数据以 SQLite 为唯一来源，前端不维护数据库副本；Tauri command 是唯一跨端调用边界。
- 当前 macOS 剪贴板窗口实际是转为 `NSPanel` 的 WebView 窗口，不是原生 AppKit 列表。
- 数据库只有一份正式 migration：`src-tauri/migrations/0001_init.sql`。FTS5 表和触发器都在该文件中，不存在 `.trellis/spec` 所述的 `0002_fts.sql`。

### 3.2 nsgo 与 AppKit

- nsgo 当前是早期项目，使用 pseudo-version，README 明确写明“no release or stable API yet”。
- nsgo 的默认 macOS 后端不是 `NSButton` / `NSTableView` / `NSTextField` 原生控件体系，而是每个节点一个翻转、layer-backed 的 `NSView`，通过 `CALayer` 和 CoreText 自绘。
- nsgo 正式提供了承载原生控件的入口：`core.Native` + `nsgo.Component` + `core.NativeFactory`。macOS 下 `NativeComponent.View()` 返回 `NSView` / `objc.ID`，宿主会把它作为 subview 加入表面。
- 因此目标架构可行，但必须采用：

```text
nsgo 只负责应用壳、窗口/Panel、托盘、快捷键、文件打开等宿主能力
        ↓
每个窗口放一个 full-size core.Native 根
        ↓
Go 通过 purego/objc（必要时 cgo bridge）构建完整 AppKit 控件树
```

- 不能把 `base` / `ui` 组件当作 AppKit 原生控件使用，也不能只依赖 nsgo 自带的表格、输入框和菜单实现来满足“真正 AppKit”的要求。

### 3.3 数据兼容

- 旧数据库、FTS5 trigram 索引、图片/图标目录和 `.ecopastebak` 都应视为已发布数据契约。
- Go 端不能对旧库重新执行 `0001_init.sql`，必须先探测 `_sqlx_migrations`、表结构和 FTS 对象，再建立自己的 schema 版本账本。
- 加密备份的容器字节布局、Argon2id 参数、XChaCha20-Poly1305 nonce/tag 顺序和 ZIP 内部路径必须逐字节兼容。
- 旧版导入目前只导入 `history` 历史记录，不导入旧设置；新实现不能把“旧版导入”误写成完整迁移。

### 3.4 最大技术风险

1. **AppKit 控件树的工程量**：`NSTableView` 数据源/委托、虚拟滚动、拖出、右键菜单、输入法、无障碍、多窗口生命周期都需要自建。
2. **nsgo 早期 API 的升级风险**：pseudo-version 无稳定承诺，必须锁定版本并隔离适配层。
3. **SQLite Go 驱动的 FTS5 trigram 兼容性**：需要在真实旧库上验证，不可仅凭驱动文档判断。
4. **macOS 时序与权限**：NSPanel key 状态、`CGEvent` 粘贴、Accessibility 授权、剪贴板 changeCount 和首次前台应用识别都不能只靠编译验证。
5. **更新链路不兼容**：当前 Tauri updater 的 endpoint、签名和包格式不能直接复用 nsgo 的 Sparkle appcast。

## 4. 分析范围与边界

### 范围内

- 当前 macOS 主路径的完整业务与数据链路。
- Windows 代码仅用于理解共享模型、数据库契约和跨平台差异，不作为第一阶段实现目标。
- nsgo 本机 module cache 中的源码与官方 gallery 示例。
- 旧数据库、旧备份和旧版 `history` 数据的兼容方案。

### 范围外

- Linux、移动端、Web 版本。
- 具体 Go 代码、AppKit 代码、SQL 迁移脚本或 UI 原型。
- 已有业务逻辑修复、依赖升级、发布流程改动。

## 5. 建议阅读顺序

1. 先读 `01` 和 `02`，建立当前功能与模块地图。
2. 再读 `03` 和 `04`，确认不可破坏的数据契约。
3. 然后读 `05`，理解 macOS 平台行为中不能简单照搬 Tauri 的部分。
4. 最后读 `06`、`07`、`08`，评估 nsgo 方案和实施顺序。

## 6. 关键源码入口

| 主题 | 源码 |
| --- | --- |
| 应用组合根与启动顺序 | `src-tauri/src/lib.rs` |
| 数据根与自定义目录 | `src-tauri/src/core/paths.rs` |
| 数据库 schema | `src-tauri/migrations/0001_init.sql` |
| 数据库连接与 migration | `src-tauri/src/db/init.rs` |
| 剪贴板读取 | `src-tauri/src/clipboard/read.rs` |
| 剪贴板建模与去重字段 | `src-tauri/src/clipboard/ingest.rs` |
| 监听闭环 | `src-tauri/src/clipboard/watcher.rs` |
| 写回与回环抑制 | `src-tauri/src/clipboard/write.rs`、`src-tauri/src/clipboard/guard.rs` |
| 图片资源 | `src-tauri/src/clipboard/storage.rs` |
| 搜索与列表查询 | `src-tauri/src/db/items.rs` |
| 设置模型与持久化 | `src-tauri/src/settings/model.rs`、`src-tauri/src/settings/store.rs` |
| 备份格式与导入 | `src-tauri/src/backup/mod.rs` |
| 旧版数据导入 | `src-tauri/src/commands/onboarding.rs` |
| macOS NSPanel | `src-tauri/src/window/macos.rs` |
| 窗口生命周期 | `src-tauri/src/window/lifecycle/mod.rs`、`descriptor.rs` |
| 快捷键 | `src-tauri/src/shortcut/mod.rs` |
| 托盘 | `src-tauri/src/tray/mod.rs` |
| 粘贴模拟 | `src-tauri/src/keystroke/macos.rs` |
| 拖出 | `src-tauri/src/drag_out/macos.rs` |
| 前端 command 边界 | `src/commands/index.ts` |
| 前端事件契约 | `src/constants/events.ts` |
| 剪贴板列表 | `src/pages/Clipboard/components/List.tsx` |
| 分页缓存 | `src/hooks/useClipboardItems.ts` |

