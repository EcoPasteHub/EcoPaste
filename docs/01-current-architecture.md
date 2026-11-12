# 当前架构与模块划分

## 1. 技术栈

| 层 | 当前实现 |
| --- | --- |
| 桌面壳 | Tauri v2 |
| 核心业务 | Rust 2021，Rust-First |
| 前端 | React 19 + React Router 7 + Ant Design v6 |
| UI 状态 | Valtio |
| 样式 | UnoCSS `presetWind4` |
| 数据库 | SQLite + sqlx 0.9，WAL |
| 剪贴板底层 | `clipboard-rs` 0.3 |
| macOS 原生交互 | `objc2`、`objc2-app-kit`、`core-graphics`、`tauri-nspanel` |
| 构建 | pnpm + Vite + Tauri CLI |
| 质量 | Biome、rustfmt、clippy、cargo test |

项目标识为 `com.ayangweb.eco-paste`，产品名为 `EcoPaste`。当前只支持 macOS 和 Windows。

## 2. 进程与启动顺序

组合根在 `src-tauri/src/lib.rs`。启动不是简单的“先建 WebView、再读设置”，而是按平台能力分阶段执行：

```text
进程启动
  -> admin::handle_startup_auto_elevation()        // Windows 自提升
  -> 初始化日志、单实例、OS、全局快捷键等 Tauri plugin
  -> macOS: 注册 tauri-nspanel plugin
  -> 注册 WindowStateStore、WindowLifecycleManager、UpdateState
  -> settings::init
  -> db::init
  -> clipboard::init
  -> shortcut::init
  -> autostart::init + sync_enabled
  -> tray::init
  -> macOS: clipboard WebView 转 NSPanel
  -> menu::clipboard_item::init
  -> 未完成 onboarding 时打开 onboarding
  -> update::schedule_auto_check
  -> Windows: 处理冷启动 `.ecopastebak` 参数
  -> RunEvent 循环
```

启动顺序中的关键依赖：

- `settings::init` 必须先于窗口、托盘、快捷键和剪贴板模块，因为它们都读取设置快照。
- `db::init` 必须先于 `clipboard::init`，监听写入依赖 `DatabaseState`。
- macOS 必须在 `to_panel` 前注册 `tauri-nspanel` plugin。
- 备份文件可能在 `setup` 完成前由 macOS `RunEvent::Opened` 投递，因此有 `APP_READY` 就绪闸和 pending slot。

## 3. Rust 模块职责

### 3.1 入口与命令层

| 模块 | 职责 |
| --- | --- |
| `lib.rs` | 插件、命令注册、启动顺序、RunEvent、单实例、文件打开、退出前保存窗口状态 |
| `commands/` | 薄 IPC 层：参数校验、调用下层服务、返回 `Result<T, AppError>` |

`commands/` 不持有业务规则，尤其是：

- 不直接写 SQL。
- 不实现剪贴板监听。
- 不决定窗口平台细节。
- 错误结构统一为 `{ kind, message }`，动作前缀由前端 toast label 提供。

主要命令域：

- 剪贴板读取、列表、预览、图片路径、文件图标、来源应用。
- 复制、粘贴、拖出、收藏、置顶、备注、分组、删除、清空。
- 窗口显示/隐藏/定位、pin、自动隐藏暂停、编辑态、窗口生命周期。
- 设置读写、重置、快捷键暂停/恢复。
- 备份导出/检查/导入、数据目录查询/切换/重置、缓存清理。
- 旧版数据检测/导入、onboarding。
- 更新检查/下载/安装/跳过版本。

### 3.2 核心业务模块

| 模块 | 主要文件 | 职责 |
| --- | --- | --- |
| `clipboard/` | `read.rs` | 用 `clipboard-rs` 读取 files/image/html/rtf/text，按采集顺序返回单一 payload |
| | `payload.rs` | 类型化剪贴板载荷 |
| | `ingest.rs` | payload -> `ClipboardItem`，类型识别、敏感检测、大小限制、摘要、图片落盘 |
| | `watcher.rs` | OS 监听线程、前台应用捕获、过滤、去重入库、事件广播 |
| | `write.rs` | text/html/rtf/image/files 写回剪贴板 |
| | `guard.rs` | 自身写回回环抑制，TTL 2 秒，按 `content_hash` 消费 |
| | `storage.rs` | 原图/缩略图分片存储 |
| | `source.rs` | macOS `NSWorkspace.frontmostApplication`；Windows 前台进程路径 |
| | `secrets.rs` | 高置信密钥/Token 检测 |
| | `apps_registry.rs` | 来源应用缓存、扫描、路径添加 |
| | `app_store.rs`、`file_icon_store.rs`、`icon.rs` | 应用图标、文件图标、图标抽取 |
| | `cleanup.rs` | 周期历史清理 |
| `db/` | `init.rs` | 连接池、WAL、外键、migration |
| | `models.rs` | 剪贴板、应用、分组、查询模型 |
| | `items.rs` | 去重、插入、查询、搜索、删除、清理 |
| | `apps.rs`、`groups.rs`、`icons.rs` | 应用、分组、文件图标仓储 |
| | `state.rs` | 可热替换的 `SqlitePool` 状态 |
| `settings/` | `model.rs` | 完整设置 schema 与默认值 |
| | `store.rs` | JSON 持久化、深度 patch、原子写 |
| `backup/` | `mod.rs` | `.ecopastebak` 导出、检查、解密、导入、文件关联接收 |
| `window/` | `mod.rs` | 统一 show/hide/toggle、布局、pin、自动隐藏暂停 |
| | `macos.rs` | NSPanel 转换、show/hide、key 状态、Dock |
| | `windows.rs` | Windows 焦点、编辑态、任务栏等 |
| | `position.rs` | 跟随光标/居中/记忆位置 |
| | `state.rs` | 窗口几何持久化 |
| | `lifecycle/` | descriptor、phase、dormant、idle destroy、keepalive |
| | `preview.rs` | 预览 overlay 窗口 |
| `menu/` | `clipboard_item.rs` | macOS 原生 muda 菜单 + Windows 菜单 payload |
| | `context_window.rs` | Windows 无焦点 WebView 菜单窗口 |
| `shortcut/` | `mod.rs` | 全局快捷键注册、冲突事件、暂停/恢复 |
| `tray/` | `mod.rs` | 菜单栏/托盘图标与菜单 |
| `drag_out/` | `macos.rs` | `NSDraggingSession` 拖出文件/图片/文本 |
| `keystroke/` | `macos.rs` | `CGEvent` 模拟 ⌘V |
| `autostart/` | | 开机自启 |
| `admin/` | | Windows 管理员启动 |
| `i18n/` | | Rust 侧托盘、菜单、命令 toast 文案 |
| `core/` | | `AppError`、路径、prevent-default、结果类型 |
| `update/` | `update.rs` | Tauri updater、通道、自动检查、跳过版本 |

### 3.3 模块依赖

```text
lib.rs
  ├─ settings ───────────────┐
  ├─ db ─────────────────────┤
  ├─ clipboard ── db/settings/core
  ├─ window ───── settings/db/platform
  ├─ shortcut ─── settings/window
  ├─ tray ─────── settings/clipboard/window
  ├─ menu ─────── db/settings/window/clipboard
  ├─ backup ───── db/settings/paths/window
  ├─ update ───── settings/window
  └─ commands ─── 以上所有服务
```

依赖方向总体是“入口 -> 服务 -> 仓储/平台适配”。当前少数模块存在相互引用，例如 tray 读取 clipboard 的 `WatcherPause`，window 读取 settings，backup 触达 window 和 settings。迁移到 Go 时应通过接口把这些依赖显式化，而不是复制模块间的隐式耦合。

## 4. 前端架构

### 4.1 路由与窗口

`src/router/index.ts` 只有一个 Hash Router，多个窗口通过 URL hash 进入不同页面：

| 路由 | 用途 |
| --- | --- |
| `/` | 剪贴板主窗口 |
| `/preference` | 偏好设置 |
| `/onboarding` | 首次启动引导 |
| `/context-menu` | Windows 列表项右键菜单窗口 |
| `/context-submenu` | Windows 二级菜单窗口 |
| `/preview` | 预览 overlay |
| `/update` | 更新窗口 |

macOS 上列表项右键菜单走原生 muda，不使用 `/context-menu`。

### 4.2 Command 与事件边界

- `src/commands/index.ts` 是唯一 Tauri command 包装入口。
- `src/constants/commands.ts` 维护命令名字面量。
- `src/constants/events.ts` 维护 Rust -> 前端事件名。
- `src/constants/windows.ts` 维护窗口 label。
- 前端禁止裸调 `invoke`、禁止散落字符串。

主要事件：

| 事件 | 含义 |
| --- | --- |
| `clipboard://updated` | 新剪贴板内容入库、去重命中、旧数据导入 |
| `clipboard-groups://updated` | 分组变化 |
| `clipboard://menu-action` | 原生菜单动作回传 |
| `settings://updated` | 设置变化 |
| `preference://highlight-setting` | 打开偏好并定位设置项 |
| `window://visibility` | 窗口显隐 |
| `window://lifecycle` | 窗口生命周期 phase |
| `window://before-destroy` | 销毁前保存草稿/申请 keepalive |
| `preview://updated` | 预览窗口状态变化 |
| `shortcut://conflict` | 全局快捷键冲突 |
| `update://progress` | 更新下载进度 |
| `backup://received` | 收到 `.ecopastebak` |
| `keyboard://nav` | Windows OS 级键盘导航 |

### 4.3 状态与数据

- Valtio 只保存 UI 状态和设置镜像。
- 业务数据从 Rust command 拉取，前端不建数据库副本。
- `src/hooks/useClipboardItems.ts` 采用每页 30、预加载 30、range cache 的分页模型。
- `src/pages/Clipboard/components/List.tsx` 负责虚拟列表、键盘导航、选中态、菜单派发和拖出触发。
- `src/pages/Preference/config/preferenceSchema.ts` 是偏好页 schema，设置项路径与 Rust `Settings` 字段对应。

### 4.4 UI 渲染

- 列表使用 `react-virtuoso`。
- HTML 预览必须经过 DOMPurify。
- RTF、Markdown、图片、文件卡片在前端渲染。
- 主题通过根 `ConfigProvider` 的 `theme.algorithm` 切换，并同步 `<html>` 的 `light` / `dark`。
- 颜色、字号、间距遵循 `AGENTS.md` 中的 antd token 和 UnoCSS 约束。

## 5. 运行时模型

### 5.1 线程

| 执行体 | 工作 |
| --- | --- |
| Tauri/AppKit 主线程 | 窗口、NSPanel、菜单、拖拽、AppKit 调用 |
| `clipboard-watcher` 线程 | `clipboard-rs` watcher 阻塞监听，平台剪贴板句柄在该线程创建 |
| Tauri async runtime | sqlx 入库、事件 emit、更新、延迟任务、监听回调投递 |
| blocking pool | 缩略图生成、部分文件 IO |

关键约束：

- `ClipboardContext` 是 `!Send`，不能在 async future 中跨 await 持有。
- `?Send` 数据的提取必须在同步段完成，再把 `Send` 数据交给 async runtime。
- AppKit/NSPanel/muda/drag 操作必须在主线程。

### 5.2 数据库连接

- 全局使用 Tauri `State<SqlitePool>`，不每次新建连接。
- 连接参数：WAL、`foreign_keys=true`、`synchronous=NORMAL`。
- 数据目录切换和 overwrite 导入会暂停 watcher，关闭旧池，热替换数据库文件后重新 `db::init`。

## 6. 当前架构中与 WebView 强耦合的部分

迁移到 AppKit 时，以下实现不能直接搬运：

| 当前能力 | WebView/Tauri 依赖 | 迁移含义 |
| --- | --- | --- |
| 页面路由 | Hash Router | 改为 AppKit 窗口/视图控制器 |
| 列表虚拟滚动 | react-virtuoso | 改为 `NSTableView` / `NSCollectionView` 复用机制 |
| 列表选中/键盘 | React 状态 + DOM 事件 | 改为 responder chain、delegate、menu validation |
| 预览 | React + HTML sanitize + DOM | 改为 `NSTextView` / `NSImageView` / WebView-free 原生渲染 |
| 右键菜单 | muda（macOS）或 WebView 窗口（Windows） | macOS 继续用 `NSMenu` 语义，不能依赖 WebView |
| command/event | Tauri IPC | 改为 Go 内部调用和 AppKit delegate/notification |
| 窗口生命周期 | Tauri WebView destroy/rebuild | 改为 NSWindowController/NSViewController 生命周期 |
| 多语言 | 前端 i18n | 可用 Go 资源 + AppKit 文案，仍需 zh-CN/en-US |

## 7. 结论

当前代码的“业务价值”主要集中在 Rust 服务和 SQLite 契约，而不是 React 页面本身。迁移时应优先保留：

1. 剪贴板采集优先级、类型识别、去重和敏感内容规则。
2. SQLite schema、FTS5 查询语义和图片资源布局。
3. `.ecopastebak` 容器与旧版历史导入。
4. macOS NSPanel 的交互语义、写回回环抑制和粘贴时序。

React 只应作为行为参考，不能作为目标 UI 的实现基础。

