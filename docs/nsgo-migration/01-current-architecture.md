# 01 · 现有项目架构与功能梳理

> 快照范围：迁移前 `master` 工作区，`package.json` 1.1.0，Tauri v2，Rust 侧约 22,580 行，前端约 20,687 行。
> 本文只描述代码中已经存在的行为，不把 README 或设计意图当成已实现能力。
> 当前仓库只保留 `legacy/backend/` 剪贴板核心与 `legacy/frontend/` UI 参考；文中 `src-tauri/...` 对应 `legacy/backend/...`，`src/...` 对应 `legacy/frontend/...`，未保留模块仅作历史描述。

## 1. 总体结构

EcoPaste 当前是典型的 Tauri v2 双进程/多 WebView 结构：

```text
┌──────────────────────────────────────────────────────────────┐
│ Tauri 主进程（Rust）                                         │
│                                                              │
│  lib.rs 启动                                                 │
│   ├─ settings::init         JSON 设置                        │
│   ├─ db::init               SQLite + FTS5                    │
│   ├─ clipboard::init        剪贴板监听 / 图标 / 清理          │
│   ├─ shortcut::init         全局快捷键                        │
│   ├─ autostart::init        开机自启                          │
│   ├─ tray::init             macOS 菜单栏 / Windows 托盘       │
│   ├─ window::*              NSPanel / 窗口生命周期 / 预览     │
│   ├─ menu::*                原生右键菜单 / Windows 菜单窗     │
│   └─ update::init           Tauri updater                    │
│                                                              │
│  命令层 commands/*  ── sqlx ── SQLite                        │
│       │                    └─ resources/ 图片、图标           │
│       └─ emit domain://action ───────────────┐               │
└──────────────────────────────────────────────┼───────────────┘
                                               │
┌──────────────────────────────────────────────▼───────────────┐
│ WebView 窗口（React 19 + Ant Design v6 + UnoCSS + Valtio）   │
│                                                              │
│  clipboard / preference / preview / onboarding / update /    │
│  context-menu / context-submenu                              │
│                                                              │
│  只渲染命令结果、维护 UI 状态、发送用户意图                    │
└──────────────────────────────────────────────────────────────┘
```

关键入口：

- `src-tauri/src/lib.rs`：插件注册、命令注册、启动顺序、窗口事件、macOS `Opened` 文件关联、单实例。
- `src-tauri/src/commands/`：Tauri command 边界，校验参数并调用下层模块。
- `src/commands/index.ts`：前端唯一的命令包装入口。
- `src/constants/commands.ts`、`src/constants/events.ts`：跨端命令名/事件名镜像。
- `src-tauri/migrations/0001_init.sql`：当前唯一已发布 migration。

## 2. 目录与职责

### 2.1 Rust 后端

| 目录 | 职责 | 迁移判断 |
| --- | --- | --- |
| `commands/` | Tauri 命令入口、参数校验、DTO 组装、事件 emit | 业务边界保留，传输层替换为 Go service/AppKit 调用 |
| `clipboard/` | 监听、读取、写回、类型识别、敏感内容、来源应用、图标、图片落盘、清理、回环抑制 | 核心业务，必须逐项迁移 |
| `db/` | sqlx 仓储、模型、FTS5 查询、迁移、测试内存库 | 核心数据层，迁到 Go + SQLite |
| `settings/` | 设置模型、默认值、校验、原子 JSON 持久化 | 核心契约，保留 JSON 形状与语义 |
| `window/` | 窗口标签、几何、NSPanel、生命周期、预览定位、轻量化 | 业务规则保留，窗口 API 换成 NSGo/AppKit |
| `shortcut/` | 全局快捷键注册、暂停/恢复、Windows Win+V | 保留行为，改用 NSGo `desktop.RegisterHotKey` |
| `tray/` | 托盘图标与菜单、暂停监听、检查更新、重启、退出 | 保留行为，改用 NSGo tray/NSMenu |
| `menu/` | 列表项右键菜单；macOS muda，Windows 独立菜单窗 | 保留菜单结构，改用 NSMenu |
| `drag_out/` | 文件/图片/文本拖出到外部应用 | 核心系统能力，必须重写为 AppKit `NSDraggingSession` |
| `backup/` | `.ecopastebak` 明文 ZIP / 加密容器、导入导出 | 核心数据契约，必须字节级兼容 |
| `i18n/` | Rust 侧用户可见文案（托盘、菜单、命令 toast） | 迁移为 Go 文案目录 |
| `autostart/` | macOS LaunchAgent / Windows 注册表与提权 | 保留行为，改用 NSGo `SetLaunchAtLogin` |
| `keystroke/` | 模拟粘贴（macOS CGEvent / Windows SendInput） | 必须重写为 AppKit/CoreGraphics |
| `keyboard/` | Windows 低级键盘钩子 | 当前 macOS 迁移不需要，保留参考 |
| `mouse/` | Windows 全局鼠标钩子，失焦隐藏 | 当前 macOS 迁移不需要，保留参考 |
| `admin.rs` | Windows 管理员启动与自提权 | 当前 macOS 迁移不需要，保留参考 |
| `update.rs` | Tauri updater、stable/beta/nightly、下载/安装/跳过 | 需要替换为 Sparkle 或等价方案 |
| `core/` | 错误、路径、prevent_default、Windows 参数 | 路径/错误模型可复用，Tauri 相关部分替换 |

### 2.2 前端

| 目录 | 职责 | 迁移判断 |
| --- | --- | --- |
| `pages/Clipboard/` | 剪贴板主窗口：搜索、分类、分组、虚拟列表、卡片、快捷动作、预览控制 | 视觉与交互参考，代码重写为 AppKit |
| `pages/Preview/` | 独立预览窗：几何、测量、动画、内容渲染 | 交互参考，重写为 AppKit 面板 |
| `pages/Preference/` | 偏好设置：schema、搜索、控件、存储、备份、权限、更新、诊断 | 设置项与文案保留为需求，控件重写为 AppKit |
| `pages/Onboarding/` | 欢迎、权限、快捷键、忽略应用、旧版导入、完成 | 流程保留，UI 重写 |
| `pages/ContextMenu/` | Windows 失焦右键菜单窗 | macOS 不需要，保留参考 |
| `pages/Update/` | 更新窗口 | 需要重新设计为 Sparkle 交互 |
| `components/` | 通用 UI 组件、图标、快捷键录制、组图标 | 大部分不复用 |
| `hooks/` | 列表分页缓存、键盘、窗口生命周期、主题、Tauri 监听 | 逻辑参考，重写为 Go/AppKit |
| `stores/` | Valtio UI 状态与设置镜像 | 用 NSGo signals 替代 |
| `locales/` | zh-CN / en-US 文案 | 文案数据可复用，加载机制重写 |
| `styles/`、`unocss/` | 全局样式、antd token 映射 | 删除/替换为 AppKit 语义色 |
| `types/` | 前后端 DTO 镜像 | 作为数据契约参考 |
| `utils/` | 日志、平台判断、快捷键、cn | 仅少量逻辑可参考 |

### 2.3 构建与发布

| 文件/目录 | 作用 | 迁移判断 |
| --- | --- | --- |
| `package.json`、`pnpm-lock.yaml`、`pnpm-workspace.yaml` | 前端依赖与脚本 | 已归档；Go 应用使用 `go.mod` |
| `vite.config.ts`、`uno.config.ts`、`tsconfig*.json` | 前端构建 | 已归档 |
| `src-tauri/Cargo.toml`、`Cargo.lock`、`build.rs`、`rust-toolchain.toml` | Rust/Tauri 构建 | 已归档；仅保留 `legacy/` 参考代码 |
| `src-tauri/tauri*.conf.json`、`capabilities/` | Tauri 窗口、打包、权限、文件关联 | 需求参考；由 `nsgo.yaml`/AppKit 替代 |
| `.github/workflows/` | PR 检查、release、nightly | 已归档；后续新建 Go/NSGo 流水线 |
| `.release-it.ts`、`CHANGELOG*.md` | 版本与变更日志 | 已归档；新发布流程另建 |
| `dist/` | 已构建的 dmg 与安装脚本 | 已删除 |
| `public/`、`src-tauri/assets/`、`src-tauri/icons/` | 图标、托盘图、音效 | 已归档；新项目资源按需重新生成 |

## 3. 核心功能清单

### 3.1 剪贴板采集

| 功能 | 现有实现 | 关键行为 |
| --- | --- | --- |
| 系统剪贴板监听 | `clipboard/watcher.rs` | macOS 轮询 `NSPasteboard.changeCount`，间隔 120ms；Windows `WM_CLIPBOARDUPDATE` |
| 文本采集 | `clipboard/read.rs`、`ingest.rs` | plain text；`content` 存源表示，`search_text` 存纯文本 |
| HTML 采集 | `read.rs`、`ingest.rs` | `content` 存 HTML 源；`search_text` 用 OS 提供的 plain 文本 |
| RTF 采集 | `read.rs`、`ingest.rs` | `content` 存 RTF 源；`search_text` 用 OS 提供的 plain 文本 |
| 图片采集 | `read.rs`、`storage.rs` | 优先 PNG 直通；非 PNG 解码后重编码；原图 + 缩略图落盘 |
| 文件/文件夹采集 | `read.rs`、`ingest.rs` | `content` 为换行分隔的绝对路径；`file_types` 为 `d/f` 紧凑串 |
| 采集优先级 | `settings::Capture.order` | 默认 `files > image > html > rtf > text`，可配置 |
| 类型开关 | `settings::Capture` | text/html/rtf/image/files 分别开关 |
| 大小上限 | `settings::Capture.max_text_mb`、`max_image_mb` | 0 表示不限；超限跳过 |
| 去重 | `db::items::upsert_item` | `blake3("<kind>:<content>")`；命中则 `use_count + 1` 并刷新 `updated_at` |
| 来源应用 | `clipboard/source.rs`、`apps_registry.rs` | 在监听回调同步抓前台应用；macOS bundle id，Windows exe 路径 |
| 来源应用图标 | `clipboard/icon.rs`、`app_store.rs` | 抽取 PNG，blake3 命名，`app-icons/` 平铺 |
| 忽略应用 | `settings::Filters.excluded_app_ids` | 命中来源应用则不入库；macOS 默认排除钥匙串与密码 App |
| 敏感内容 | `clipboard/secrets.rs` | 私钥、常见 token、AWS key、JWT、带标签 secret；可配置是否收录/是否脱敏 |
| 回环抑制 | `clipboard/guard.rs` | 写回前登记 content hash，2 秒 TTL，监听侧命中则跳过 |
| 监听暂停 | `WatcherPause` + 托盘菜单 | 暂停后跳过整条入库链路，但不停止 watcher 线程 |
| Windows 读重试 | `watcher.rs` | 15/35/75ms 三次重试，应对其他监听器短暂占用 |
| 复制音效 | `clipboard/sound.rs` | `rodio` 播放 `assets/sounds/copy.mp3` |
| 手动读取 | `commands::read_clipboard` | 与自动采集共用 `build_item_with_settings` 和 `persist_and_notify` |

### 3.2 历史数据与存储

| 功能 | 现有实现 | 关键行为 |
| --- | --- | --- |
| SQLite 主库 | `db/init.rs`、`db/path.rs` | `<data_root>/db/clipboard.db`；WAL、foreign_keys、Normal synchronous |
| Migration | `src-tauri/migrations/0001_init.sql` | sqlx `migrate!`；已发布 migration 不回改 |
| 全文搜索 | `0001_init.sql`、`db/items.rs` | FTS5 `trigram`；≥3 字符走 FTS，1–2 字符走 `LIKE` 兜底 |
| 历史查询 | `db/items.rs` | kind/group/favorite/pinned/group filter/keyword/sort/limit/offset |
| 分页 | `ClipboardItemPage` | `list` + `total` + `has_more`，一次命令返回 |
| 收藏 | `toggle_item_favorite`、`mark_item_favorite` | `is_favorite`；auto-favorite 可配置 |
| 置顶 | `toggle_item_pinned` | `is_pinned`；置顶项永远排在普通项前 |
| 备注 | `update_item_note` | 不刷新 `updated_at` |
| 自定义分组 | `db/groups.rs` | 名称、SVG 图标、隐藏、排序；删除时条目 `group_id` 置 NULL |
| 分组布局 | `update_clipboard_groups_layout` | 批量更新 `sort_order` |
| 使用次数 | `increment_item_use_count` | `use_count + 1`，并刷新 `updated_at` |
| 历史清理 | `clipboard/cleanup.rs`、`db/items.rs` | 按保留时长 + 最大条数；收藏/置顶保留；启动先跑一次 |
| 清空记录 | `clear_items` | 可配置是否保留收藏/置顶 |
| 数据根 | `core/paths.rs` | `<app_local_data>/<dev\|prod>`；`storage.json` 指向真实根 |
| 自定义数据目录 | `commands/storage.rs` | 迁移 `db/resources/config/state`，热切换连接池 |
| 资源目录 | `core/paths.rs` | `resources/clipboard-images/{origin,thumbnails}`、`app-icons`、`file-icons` |
| 存储占用 | `get_storage_usage` | total/database/resources/settings 分项 |
| 缓存清理 | `clean_resource_cache` | 删除未被 DB 引用的图片/应用图标/文件图标 |
| 旧版数据导入 | `commands/onboarding.rs` | 扫描 v0.6 `history` 表与 `images/`，按普通/收藏选择性导入 |

### 3.3 内容识别与预览

| 功能 | 现有实现 | 关键行为 |
| --- | --- | --- |
| URL | `clipboard/detect.rs` | 必须带协议或 `www.`，单行 |
| Email | `detect.rs` | RFC 常见字符，支持中文 |
| 颜色 | `detect.rs` | hex、rgb/hsl/hwb/lab/lch/oklab/oklch/color/color-mix、渐变；输出可信 CSS |
| 路径 | `detect.rs` | 仅绝对路径且存在 |
| 列表摘要 | `ingest.rs` | trim 后最多 256 字符 |
| 列表富文本降级 | `commands/clipboard.rs` | 列表只返回 `summary`，不把完整 HTML/RTF 送前端 |
| 图片缩略图 | `storage.rs` | 最长边 300px，懒生成，按 hash 分片 |
| 文件条目 | `commands/clipboard.rs` | 路径、名称、目录/图片标记、存在性、图标、大小 |
| 文件预览模式 | `FilesPreviewKind` | 单文件图片走 `ImagePreview`，其余走 `List` |
| 预览窗口 | `window/preview.rs`、`src/pages/Preview/` | hover / Space 触发；独立窗口；几何测量与动画 |
| HTML/RTF 预览 | `preview_text()` | 目前预览使用 OS 提供的纯文本，不在预览窗渲染富文本源 |

### 3.4 历史复用动作

| 功能 | 现有实现 | 关键行为 |
| --- | --- | --- |
| 复制 | `clipboard/write.rs` | 按 kind 写回 text/html/rtf/image/files；可强制纯文本 |
| 粘贴 | `commands::paste_clipboard_item` | 写回 + 隐藏/让出 key + 模拟 ⌘V；固定窗口时 resign/restore key |
| 纯文本粘贴 | `write.rs` + `paste_plain` | 文本只写 plain；HTML/RTF 同时写 plain 回退 |
| 文件路径粘贴 | `paste_files_as_path` | 文件记录写回换行路径文本 |
| 打开链接/邮件 | `open_clipboard_item_link` | URL / email 子类型 |
| Finder/资源管理器显示 | `reveal_clipboard_item` | macOS Finder / Windows Explorer |
| 图片另存 | `save_clipboard_image_to_file` | 默认文件名 `EcoPaste-image-YYYYMMDD-HHMMSS.png` |
| 拖出 | `commands/drag.rs`、`drag_out/*` | 文件、图片、文本（plain/HTML/RTF）；macOS `NSDraggingSession` |
| 备注编辑 | `NoteModal`、`update_clipboard_item_note` | 列表内联备注标注 |
| 收藏/置顶 | 快捷动作、右键菜单 | 有独立删除保护设置 |
| 移动到分组 | 右键菜单子菜单 | 分组列表实时读取 |
| 删除 | `delete_clipboard_item` | 图片记录同时删除落盘文件 |
| 自动粘贴 | `settings::Content.auto_paste` | 单击/双击、粘贴/复制、禁用 |
| 中键动作 | `settings::Content.middle_click` | 粘贴/纯文本粘贴/复制/纯文本复制/禁用 |
| 悬停快捷动作 | `settings::Content.item_actions` | 可配置、可排序 |

### 3.5 窗口与系统集成

| 功能 | 现有实现 | 关键行为 |
| --- | --- | --- |
| 剪贴板窗口 | `window/mod.rs`、`window/macos.rs` | macOS NSPanel，非激活、可成为 key、置顶、无边框、透明 |
| 窗口定位 | `window/position.rs` | 跟随鼠标 / 居中 / 记住位置 |
| 失焦隐藏 | `window/macos.rs`、Windows mouse hook | 保存编辑状态、pin、自动隐藏暂停 |
| 固定窗口 | `set_clipboard_window_pinned` | 固定后不自动隐藏；粘贴时 resign/restore key |
| 窗口生命周期 | `window/lifecycle/*` | ready/visible/hiddenWarm/dormant/destroyPending/destroyed；轻量化释放 WebView |
| 窗口几何持久化 | `window/state.rs` | `<data_root>/state/window-state.json` |
| 偏好窗口 | `window/mod.rs`、`pages/Preference` | 独立窗口；支持定位到指定设置项 |
| 预览窗口 | `window/preview.rs` | 独立窗口；不抢焦点；跟随剪贴板窗口位置 |
| 更新窗口 | `window/mod.rs`、`pages/Update` | 检查/下载/安装/跳过 |
| 引导窗口 | `window/mod.rs`、`pages/Onboarding` | 首次启动；6 步流程 |
| 托盘 | `tray/mod.rs` | 偏好、暂停/开始监听、开源地址、检查更新、版本、重启、退出 |
| 全局快捷键 | `shortcut/mod.rs` | 唤起剪贴板 / 偏好；录入时暂停；冲突事件 |
| Win+V | `shortcut/win_v.rs` | Windows 专用，替代系统剪贴板面板 |
| 开机自启 | `autostart/*` | macOS LaunchAgent，Windows 注册表 |
| Dock/任务栏图标 | `show_taskbar_icon` | macOS Dock / Windows 任务栏显隐 |
| 单实例 | `tauri-plugin-single-instance` | 第二实例唤起偏好/接收备份；自启并发不弹窗 |
| 文件关联 | `tauri.conf.json`、`backup/*` | `.ecopastebak`；macOS `Opened`，Windows 启动参数/第二实例 |
| 原生右键菜单 | `menu/clipboard_item.rs` | macOS `muda` NSMenu；动作分组、快捷键、子菜单 |
| Windows 菜单窗 | `menu/context_window.rs` | 避免 `TrackPopupMenu` 抢焦点 |
| 权限 | `tauri-plugin-macos-permissions`、`PermissionControl` | 辅助功能、完全磁盘访问、Windows 管理员 |
| 管理员启动 | `admin.rs` | Windows UAC 自提权 |
| 防止默认行为 | `core/prevent_default.rs` | 生产环境禁用 webview 默认右键/刷新等 |
| 更新 | `update.rs` | Tauri updater；stable/beta/nightly endpoint；下载进度事件 |

### 3.6 UI 页面与交互

| 页面 | 主要能力 |
| --- | --- |
| Clipboard |  Header 搜索、pin/unpin、清空、打开偏好；分类 Tab（全部/文本/图片/文件/收藏）；自定义分组；虚拟列表；文本/图片/文件卡片；颜色预览；来源应用图标；备注；悬停快捷动作；右键菜单；键盘导航；Footer 总数与快捷键 |
| Preview |  文本/图片/文件预览；hover 与 Space；边缘定位；连接线动画；内容切换；最大 480×480；缓存 16 条 |
| Preference |  记录/整理/复用/工作流/快捷键/数据/关于 7 个 Tab；搜索设置；设置项高亮；存储占用；数据目录；缓存清理；备份导入导出；分组管理；来源应用忽略；权限；主题/语言；更新设置；诊断；重置 |
| Onboarding |  Welcome → Permissions → Shortcuts → Ignore Apps → Legacy Import → Done；可回退；记录 `lastStep`；完成打开剪贴板 |
| Update |  当前版本、检查、下载进度、安装、跳过版本、频道说明 |
| ContextMenu |  Windows 专用；列表项右键菜单与分组子菜单 |

### 3.7 国际化与主题

- 语言：`zh-CN`（默认）与 `en-US`；`src/locales/` 前端文案，`src-tauri/src/i18n/` 托盘/菜单/命令短文案。
- 主题：`auto/light/dark`；根部 `ConfigProvider` + `<html class="light|dark">`；`useAppTheme` 同步 Tauri 窗口主题。
- 颜色：antd token 映射到 UnoCSS；自定义组件使用 `text-ant-*`、`bg-ant-*`。
- macOS 原生 chrome：NSPanel、菜单、托盘、文件面板、Finder reveal、权限弹窗。

## 4. 代码分类

### 4.1 核心业务（必须保留/迁移）

- 剪贴板采集与写回：`clipboard/{watcher,read,write,ingest,payload,detect,secrets,guard,source,storage,app_store,apps_registry,file_icon_store,cleanup,sound}.rs`
- 数据层：`db/{models,items,groups,apps,file_icons,init,state,path}.rs`、`migrations/0001_init.sql`
- 设置：`settings/{model,store}.rs`
- 备份：`backup/mod.rs`
- 存储路径与迁移：`core/paths.rs`、`commands/storage.rs`
- 窗口业务规则：`window/{mod,position,state,preview}.rs`、`window/lifecycle/*`
- 系统集成：`shortcut/mod.rs`、`tray/mod.rs`、`menu/*`、`drag_out/*`、`keystroke/*`、`autostart/*`、`update.rs`
- 旧版导入：`commands/onboarding.rs`
- 数据契约：`db/models.rs`、`settings/model.rs`、`src/types/*`、`src/constants/*`

### 4.2 UI 层（迁移时重写）

- `src/pages/*`、`src/components/*`、`src/hooks/*`、`src/stores/*`、`src/styles/*`、`src/unocss/*`
- Ant Design、UnoCSS、motion、Valtio、react-router、react-virtuoso、ahooks、i18next/react-i18next 的使用
- Tauri WebView 专属能力：`getCurrentWebviewWindow()`、`convertFileSrc`、`useTauriListen`、`invoke`、WebView 生命周期

### 4.3 基础设施（迁移时替换）

- Tauri v2 runtime、plugins、capabilities、`tauri.conf.json`
- Vite、pnpm、Node 脚本、Biome、lint-staged、simple-git-hooks
- Cargo/rustfmt/clippy、Tauri updater
- `dist/`、`src-tauri/target/`（如存在）、`node_modules/`（如存在）等构建产物

### 4.4 平台原生代码（按 macOS 优先级重写）

- macOS：`window/macos.rs`、`drag_out/macos.rs`、`keystroke/macos.rs`、`autostart/macos.rs`、`clipboard/source.rs` 的 macOS 分支、`clipboard/icon.rs`
- Windows：`window/windows.rs`、`drag_out/windows*.rs`、`keyboard/windows.rs`、`mouse/windows.rs`、`keystroke/windows.rs`、`shortcut/win_v.rs`、`autostart/windows.rs`、`admin.rs`
- 当前 macOS 迁移只要求重写 macOS 分支；Windows 代码保留作后续参考，不新增跨平台抽象。

### 4.5 第三方依赖（逐项评估）

Rust 侧：

- `clipboard-rs`：OS 剪贴板读写与 watcher。NSGo 只提供 text/image/files 基础服务；HTML/RTF、watcher、文件写入需要 AppKit 直连或替换库。
- `sqlx`：迁移到 `modernc.org/sqlite`（纯 Go、无 CGO）或 `zombiezen.com/go/sqlite`。
- `blake3`、`argon2`、`chacha20poly1305`、`zip`、`walkdir`、`image`、`rodio`、`file_icon_provider`、`drag`、`auto-launch`、`tauri-plugin-*`：逐项寻找 Go 等价物或 AppKit 原生实现。
- `objc2` / `objc2-app-kit`：迁移为 NSGo 使用的 `purego/objc` 或 `github.com/ebitengine/purego`。

前端：

- `antd`、`ahooks`、`motion`、`overlayscrollbars`、`react-virtuoso`、`valtio`、`react-router`、`i18next`、`dompurify`、`tailwind-merge`、`clsx`：迁移后不再使用；只保留文案、图标和交互参考。

## 5. 数据契约

### 5.1 SQLite schema

`src-tauri/migrations/0001_init.sql` 当前定义：

- `clipboard_groups(id, name, icon, is_hidden, sort_order, created_at, updated_at)`
- `clipboard_apps(id, name, icon_file, platform, created_at, updated_at)`
- `clipboard_items(id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text, summary, file_types, size, width, height, use_count, is_favorite, is_pinned, is_sensitive, platform, note, created_at, updated_at)`
- `file_type_icons(cache_key, platform, icon_file, created_at, updated_at)`
- `clipboard_items_fts`：FTS5，`search_text` + `note`，`content='clipboard_items'`，`tokenize='trigram'`
- 三个触发器：insert / delete / update 同步 FTS
- 索引：`content_hash`、`source_app_id`

已发布数据约束：

- `created_at` / `updated_at` 为 `TEXT NOT NULL`。
- `updated_at` 表示“内容再次使用时间”，备注/收藏/置顶/分组变更不得刷新。
- `clipboard_items.kind` 为 `text|image|files`；`sub_kind` 为 `rtf|html|url|email|color|path`。
- `platform` 为 `macos|windows`。
- 文件记录 `content` 为换行分隔路径，`file_types` 为 `d/f` 逗号串。
- 图片记录 `content` 为 `<blake3>.png` 文件名，实际文件在 `resources/clipboard-images/origin/<hash[..2]>/`。
- `source_app_id` 外键 `ON DELETE SET NULL`；`group_id` 外键 `ON DELETE SET NULL`。

### 5.2 设置 JSON

`settings/model.rs` 定义，落盘 `<data_root>/config/settings.json`，全部 `serde(default)`、`rename_all = "camelCase"`。

顶层：

```text
general, appearance, shortcuts, clipboard, onboarding, update
```

关键子结构：

- `general`: `autoStart`, `runAsAdmin`, `trayIcon`, `dockIcon`
- `appearance`: `theme` (`auto|light|dark`), `language` (`zh-CN|en-US`)
- `shortcuts`: `openClipboard`, `openPreference`, `winV`
- `clipboard.capture`: `text/html/rtf/image/files`, `maxTextMb`, `maxImageMb`, `order`
- `clipboard.content`: `autoPaste`, `middleClick`, `copyPlain`, `copyThenHideWindow`, `pastePlain`, `pasteFilesAsPath`, `showOriginalPreview`, 删除保护, `autoFavorite`, `updateOnReuse`, `sort`, `itemActions`, `itemActionOrder`
- `clipboard.display`: `textMaxLines`, `imageMaxHeight`, `fileMaxCount`
- `clipboard.sensitive`: `collectSecrets`, `redactSecrets`
- `clipboard.history`: `retention{value,unit}`, `maxCount`, `cleanupIntervalHours`
- `clipboard.search`: `defaultFocus`, `clearOnHide`
- `clipboard.window`: `position`, `scrollToTopOnOpen`, `selectRangeOnOpen`, `selectCategoryOnOpen`, `selectGroupOnOpen`, `lightweightMode`, `idleDestroySeconds`
- `clipboard.preview`: `hoverEnabled`, `hoverDelayMs`, `spaceEnabled`
- `clipboard.feedback`: `copySound`
- `clipboard.filters`: `excludedAppIds`
- `onboarding`: `completed`, `lastStep`, `legacyImport{checked,imported,importTypes,importedAt}`
- `update`: `autoCheck`, `frequency`, `includeBeta`, `includeNightly`, `lastCheckedAt`, `skippedVersion`

写入语义：

- `update_settings` 只接受 JSON object patch。
- 对象深度合并；数组整体替换。
- 校验跨字段约束（快捷键不能重复、`selectGroupOnOpen` 编码合法）。
- 先写 `.tmp` 再 rename，原子替换。

### 5.3 数据目录

```text
~/Library/Application Support/com.ayangweb.eco-paste/
  <dev|prod>/
    storage.json                 # bootstrap manifest，指向真实数据根
    db/clipboard.db              # SQLite + -wal/-shm
    resources/
      clipboard-images/
        origin/<2hex>/<hash>.png
        thumbnails/<2hex>/<hash>.png
      app-icons/<hash>.png
      file-icons/<hash>.png
    config/settings.json
    state/window-state.json
```

自定义数据目录：

```text
<user-selected-parent>/EcoPasteData/<dev|prod>/
  .ecopaste-storage.json
  db/ resources/ config/ state/
```

旧版 v0.6 数据目录（onboarding 扫描）：

```text
~/Library/Application Support/com.ayangweb.EcoPaste/
  EcoPaste.db / EcoPaste.dev.db
  store.json / store.dev.json
  images/
```

### 5.4 备份格式

`backup/mod.rs`：

- 扩展名 `.ecopastebak`。
- 明文：标准 ZIP，包含 `manifest.json`、`db/clipboard.db`、`resources/**`、`config/settings.json`。
- 加密：自定义容器：
  - magic `ECOPASTEBAK1`（12 字节）
  - 4 字节 little-endian header 长度
  - JSON header：`formatVersion`, `mode`, `kdf{algorithm,memoryKib,timeCost,parallelism,salt}`, `cipher{algorithm,nonce}`
  - `Argon2id`，`memoryKib=65536`, `timeCost=3`, `parallelism=1`, salt 16 字节
  - `ChaCha20-Poly1305`, nonce 24 字节
- 导入策略：`merge` / `overwrite`。
- 接收来源：`openFile` / `dragDrop`；偏好窗销毁时用 pending slot 兜底。

### 5.5 命令与事件

命令约 90 个，集中在 `src-tauri/src/lib.rs` 的 `generate_handler!` 与 `src/commands/index.ts`。主要域：

- 剪贴板：读取、列表、单条、预览、写回、粘贴、拖出、收藏/置顶/备注/分组/删除/清空/打开链接/reveal/保存图片
- 分组：列表、创建、更新、布局、删除、SVG 导入
- 来源应用：列表、全部应用、从路径添加、删除未引用
- 设置：读取、更新、重置、暂停/恢复快捷键
- 窗口：show/hide/toggle/ready/dirty/keepalive/lifecycle/pin/auto-hide/editing/preview
- 存储：位置、占用、切换、重置、清缓存、打开目录
- 备份：导出、检查、导入、pending
- 引导：打开、步骤、完成、检测/导入旧数据
- 更新：状态、检查、下载、安装、跳过、打开更新窗
- 其他：自启、管理员、外部链接、复制音效、文件图标、应用图标

事件名统一 `domain://action`，见 `src/constants/events.ts`：

```text
backup://received
clipboard-groups://updated
clipboard://menu-action
clipboard://updated
context-menu://show
context-submenu://show
keyboard://nav
preference://highlight-setting
preview://updated
settings://updated
update://progress
window://before-destroy
window://lifecycle
window://visibility
```

## 6. 现有实现的隐藏耦合

这些是迁移时最容易漏掉的部分：

1. **命令层会“补字段”**：`list_clipboard_items` 不只查 DB，还会补缩略图绝对路径、来源应用图标路径、文件条目、颜色预览、展示时间、脱敏摘要、可用动作。NSGo 侧不能只复刻 SQL。
2. **列表与预览的 content 裁剪不同**：列表查询把 text 的 `content/search_text` 置空，只给 `summary`；预览/写回才查完整 `content`。迁移时要保留这个性能边界。
3. **`updated_at` 语义特殊**：它不是普通 `updated_at`，而是“最近一次复用时间”；笔记/收藏/置顶/分组不能刷新。
4. **FTS5 trigram 最短 3 字符**：1–2 字符关键词必须走 `LIKE`，否则搜索永远为空。
5. **写回与监听的哈希必须完全一致**：否则回环抑制失效，点击粘贴会新增一条历史。
6. **来源应用必须在监听回调同步抓取**：延后到异步入库时前台应用已经切走。
7. **macOS panel 的 key window 语义**：粘贴前必须 hide 或 resign key，等 50ms 再发 ⌘V；固定窗口还要 restore key。
8. **窗口轻量化会销毁 WebView**：偏好/更新等窗口有 pending slot 和重建机制；原生窗口不需要 WebView 重建，但事件时序仍需重新设计。
9. **右键菜单不是前端组件**：macOS 用 `muda` NSMenu，Windows 用独立 webview 窗；迁移到 NSMenu 后，动作执行仍在业务层。
10. **备份/设置/数据库是已发布数据契约**：不是内部实现细节，必须字节级兼容或提供迁移。

## 7. 迁移视角下的模块优先级

| 优先级 | 模块 | 理由 |
| --- | --- | --- |
| P0 | 数据根、设置、SQLite schema、content hash、备份格式 | 一旦破坏会丢用户数据 |
| P0 | 剪贴板采集/写回/去重/回环抑制/来源应用 | 产品核心，错误会直接污染历史 |
| P0 | macOS 面板、全局快捷键、托盘、粘贴模拟 | 日常主路径 |
| P1 | 列表查询、搜索、分组、收藏/置顶/备注、清理 | 历史管理主路径 |
| P1 | 预览、图片/文件图标、拖出 | 高频但可在 parity 阶段补齐 |
| P1 | 偏好设置、权限、引导、旧版导入 | 首次启动与可配置性 |
| P2 | 更新、备份 UI、诊断、音效、主题细节 | 可在核心稳定后补齐 |
| P2 | Windows 专用能力 | 当前 macOS 迁移不要求 |

## 8. 迁移前测试与验证基线

> 以下是迁移前记录，当前仓库已移出这些工程，不能直接执行；保留用于说明旧版验证范围。

- Rust：`cargo fmt`、`cargo clippy -- -D warnings`、`cargo test`；剪贴板真实系统测试标记 `#[ignore]`。
- 前端：`pnpm lint`、`pnpm tsc`；没有前端测试 runner。
- CI：`.github/workflows/pr-check.yml` 分前端/Rust 两条；release/nightly 走 Tauri 打包。
- 迁移后必须新增：
  - 数据兼容测试（用真实 DB 副本打开、查询、写入、FTS、备份导入导出）。
  - 剪贴板格式矩阵测试（plain/HTML/RTF/image/files + 多格式同时存在）。
  - AppKit 组件测试（`nsgotest` + 人工 UI 验收）。
  - 备份格式 round-trip（明文/加密，与旧版文件互操作）。
