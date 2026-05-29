# EcoPaste 重构 TODO

> 参考项目（旧版，只读参考，**不要修改**）：
>
> - 公开仓库：https://github.com/EcoPasteHub/EcoPaste
> - 本地副本（可选加速）：`/Users/ayang/Documents/PersonalProject/2024/EcoPaste_bak` — 若已 clone 到此路径，能访问文件系统的 AI 工具优先本地读取（可直接 grep/遍历目录，快于逐个远程 fetch）。
>
> 目标技术栈：**Tauri v2 + React 19 + HeroUI v3 + TailwindCSS v4**
> 核心原则：**逻辑尽量下沉 Rust，前端只做 UI 展示与交互**；仅当 Rust 不适合时才放前端。

## 技术选型（已确认）

| 维度        | 决策                   | 说明                                                           |
| ----------- | ---------------------- | -------------------------------------------------------------- |
| 代码组织    | **当前项目新建空分支** | `pnpm create tauri-app` 一键生成脚手架，逐功能从旧项目迁移参考 |
| 桌面框架    | Tauri v2               | `tray-icon` / `protocol-asset` / `macos-private-api`           |
| 前端框架    | React 19               | 新特性：Actions、`use`、`useOptimistic`、ref as prop           |
| UI 组件库   | HeroUI v3              | 替换旧项目的 Ant Design 5                                      |
| 样式        | TailwindCSS v4         | 替换旧项目的 UnoCSS；CSS-first 配置（`@theme`）                |
| Rust 数据层 | **sqlx (SQLite)**      | 异步 + 编译期 SQL 校验 + 内置 migration                        |
| 前端状态    | **Valtio**             | 仅管理 UI 状态，业务状态来自 Rust                              |
| 构建        | Vite + pnpm            |                                                                |
| Lint/Format | Biome                  |                                                                |
| 范围策略    | **MVP 先行**           | 阶段 0–7 为 MVP，阶段 8+ 为增强                                |
| 支持平台    | **仅 macOS + Windows** | 不支持 Linux；平台特化用 `#[cfg(target_os)]` 隔离              |

## Rust-First 职责划分

> 这是本次重构的灵魂。明确每块逻辑归属，避免迁移中反复横跳。

**下沉到 Rust（核心）**

- 剪贴板监听（OS 级监听，替代旧项目的前端轮询/回调）
- 数据库全部读写（sqlx，前端不再直连 SQL，旧项目用 Kysely 直查）
- 内容类型识别（URL / email / color / path 的检测）
- 历史记录清理（保留时长 / 最大条数的后台任务）
- 全文搜索（SQLite FTS5）
- 写回剪贴板 + 模拟粘贴（已有 eco-paste 思路）
- 窗口定位计算（跟随光标 / 居中 / dock 的坐标数学）
- 图片落盘与缩略图、文件元信息读取
- 设置项持久化（统一 store，由 Rust 落盘）

**保留在前端（UI 必需）**

- 组件渲染、虚拟滚动、瀑布流布局
- 主题切换的视觉应用、CSS 变量注入
- i18n 文案渲染（文案表前端持有，语言选择可由 Rust 提供系统语言）
- HTML 预览的 DOM 渲染与 sanitize（DOMPurify）、RTF 渲染、Markdown 渲染
- 键盘交互、列表选中态、动画

---

# 阶段 0 · 项目脚手架与基础设施

### 0.1 Tauri v2 脚手架

- [x] 更新 Rust 工具链（`rustup update`），按需补齐交叉编译 target（aarch64/x86_64-apple-darwin、x86_64-pc-windows-msvc）
- [x] 在 `next` 分支用 `pnpm create tauri-app` 生成 React + TS + Vite 模板（自带 `.gitignore` / `package.json` / `README` 等）
- [x] 校验 `pnpm tauri dev` 能跑通空白窗口
- [x] 锁定 pnpm 版本（`packageManager` 字段）与 Rust 版本（`rust-toolchain.toml`）
- [x] 配置 `tsconfig.json`（strict、paths 别名 `@/*`）

### 0.2 前端 UI 基础

- [x] 安装并配置 TailwindCSS v4（`@tailwindcss/vite` 插件 + `@import "tailwindcss"`）
- [x] 安装 HeroUI v3（`@heroui/react` + `@heroui/styles`），CSS 接入 `@import "@heroui/styles"` + `@custom-variant dark`
- [x] 验证 Tailwind v4 与 HeroUI v3 的样式无冲突（层级 `@layer`）
- [x] 安装路由 react-router（main `/` 与 preference `/preference` 两路由）

### 0.3 代码规范

- [x] 配置 Biome（`biome.json`：格式化 + lint 规则）
- [x] 配置 commitlint + simple-git-hooks + lint-staged（pre-commit 跑 Biome）
- [x] Rust 侧配置 `rustfmt.toml` 与 `clippy`（CI 中 `cargo clippy -- -D warnings`）

### 0.4 Rust 项目骨架

- [ ] 规划 `src-tauri/src` 模块目录：`commands/` `db/` `clipboard/` `window/` `paste/` `settings/` `core/`（platform setup）
- [ ] `lib.rs` 搭建 Tauri `Builder` 骨架，`main.rs` 调用 `lib::run()`
- [ ] 引入 `serde` / `serde_json` / `thiserror` / `anyhow`（错误处理）
- [ ] 定义统一错误类型 `AppError` + `Result<T>`，并实现 `serde::Serialize` 供命令返回
- [ ] 引入 `tracing` 或 `tauri-plugin-log` 做日志

---

# 阶段 1 · 数据层（Rust + sqlx）★ 核心

### 1.1 数据库初始化

- [ ] 引入 `sqlx`（features：`runtime-tokio` / `sqlite` / `macros` / `chrono`）
- [ ] 确定 DB 路径：app data dir 下 `EcoPaste.db`（dev 用 `EcoPaste.dev.db`），由 Rust 解析路径
- [ ] 实现 `db::init()`：创建/打开数据库 + `SqlitePool` 连接池，存入 Tauri `State`
- [ ] 启用 SQLite pragma（`journal_mode=WAL`、`foreign_keys=ON`、`synchronous=NORMAL`）

### 1.2 Migration（编译期校验）

- [ ] 建立 `migrations/` 目录，配置 `sqlx::migrate!`
- [ ] 编写 `0001_init.sql`：`history` 表
  - 字段：`id TEXT PK`、`type`、`subtype`、`group`、`value`、`search`、`count INTEGER`、`width`、`height`、`favorite INTEGER`、`note`、`create_time`
  - 类型枚举：`text` / `rtf` / `html` / `image` / `files`；text 子类型：`url` / `email` / `color` / `path`
- [ ] 编写 `0002_fts.sql`：FTS5 虚表 `history_fts` + 触发器（insert/update/delete 同步）
- [ ] 启动时自动执行 migration，失败时优雅报错
- [ ] 配置 `.env`（`DATABASE_URL`）以启用 sqlx 编译期校验（或离线模式 `sqlx prepare` 生成 `.sqlx`）

### 1.3 数据模型与仓储层

- [ ] 定义 Rust 结构体 `HistoryItem`（`#[derive(Serialize, Deserialize, FromRow)]`）
- [ ] 定义查询参数结构体（分页、过滤 type、favorite、group、关键词）
- [ ] 实现仓储函数（全部 async）：
  - [ ] `insert_history(item)`
  - [ ] `query_history(filter, page, size)` — 分页 + 排序（时间/频率）
  - [ ] `search_history(keyword, filter)` — 走 FTS5
  - [ ] `update_history(id, patch)` — 收藏 / 备注 / count++
  - [ ] `delete_history(id)` / `delete_batch(ids)` / `clear_all(keep_favorite: bool)`
  - [ ] `get_by_id(id)`
  - [ ] `toggle_favorite(id)`
- [ ] 写仓储层单元测试（用内存库 `sqlite::memory:`）

### 1.4 去重与计数逻辑（Rust）

- [ ] 入库前对相同内容做哈希/比对，命中则 `count++` 并刷新 `create_time`（替代旧前端逻辑）
- [ ] 选择哈希策略（text 直接比对；image/files 用内容哈希）

---

# 阶段 2 · 剪贴板监听与内容处理（Rust）★ 核心

### 2.1 剪贴板读取插件

- [ ] 评估 `tauri-plugin-clipboard-x` 复用 vs 自研（旧项目用了自定义 fork）
- [ ] 引入剪贴板能力，支持读取：纯文本 / HTML / RTF / 图片 / 文件列表
- [ ] 封装 `clipboard::read_all() -> ClipboardPayload`（带类型标记）

### 2.2 OS 级监听（替代前端轮询）★

- [ ] macOS：基于 `NSPasteboard` 的 `changeCount` 轮询线程（或事件）
- [ ] Windows：`AddClipboardFormatListener` 监听 `WM_CLIPBOARDUPDATE`
- [ ] 统一抽象 `ClipboardWatcher`，变更时触发回调
- [ ] 监听线程产出变更 → 经内容处理 → 入库 → `emit` 事件通知前端刷新
- [ ] 处理自身写回剪贴板导致的回环（写回时打标记/抑制下一次监听）

### 2.3 内容类型识别（Rust）★

- [ ] 文本子类型检测：`url`（含 `is-url` 等价正则）/ `email` / `color`（hex/rgb/hsl）/ `path`
- [ ] 生成 `search` 字段：HTML→纯文本、RTF→纯文本、image→占位、files→路径串
- [ ] 图片处理：解码取 `width`/`height`，落盘到 app data `images/`，`value` 存引用
- [ ] 文件处理：读取路径列表、文件大小/类型元信息
- [ ] 单元测试覆盖各类型识别

### 2.4 暴露命令

- [ ] `#[tauri::command]` 包装 2.1–2.3，供前端按需手动触发（如「重新读取」）

---

# 阶段 3 · 窗口管理（Rust + 平台特化）★ 核心

### 3.1 窗口配置

- [ ] `tauri.conf.json` 定义两个窗口：
  - `main`：360×600、透明、无边框、置顶、跳过任务栏、默认隐藏、`#/`
  - `preference`：700×480、居中、可调、`#/preference`
- [ ] 配置 capabilities/permissions（v2 权限模型，最小授权）

### 3.2 自定义窗口插件（eco-window 思路）

- [ ] 命令：`show_window` / `hide_window` / `toggle_window` / `show_taskbar_icon`
- [ ] 窗口定位计算（Rust）★：跟随光标 / 屏幕居中 / dock 底部；多显示器下取光标所在屏
- [ ] 窗口位置/尺寸持久化（由 Rust 落盘 `.window-state.json`），启动恢复

### 3.3 macOS NSPanel 特化

- [ ] 引入 `tauri-nspanel`，将 main 窗口转为 NSPanel（浮层、dock level）
- [ ] 隐藏 dock 图标、可在全空间显示、`acceptsFirstMouse`
- [ ] 绑定 focus/blur/resize/move 事件 → emit 给前端
- [ ] 失焦自动隐藏（可配置）

### 3.4 关闭行为

- [ ] `WindowEvent::CloseRequested`：隐藏而非退出（常驻后台）
- [ ] macOS 应用 reopen 事件处理

---

# 阶段 4 · 粘贴与写回剪贴板（Rust）★ 核心

### 4.1 写回剪贴板

- [ ] `write_to_clipboard(item)`：按类型写回 text/rtf/html/image/files
- [ ] 纯文本模式：剥离格式只写 plain text

### 4.2 模拟粘贴（eco-paste 思路）

- [ ] macOS：`cocoa` 触发 Cmd+V
- [ ] Windows：`enigo` / `winapi` 模拟 Ctrl+V
- [ ] `paste(item, plain: bool)` 组合命令：写回 + 触发粘贴
- [ ] 粘贴前隐藏主窗口、聚焦回原应用的时序处理

---

# 阶段 5 · 全局快捷键与自启动（Rust）

### 5.1 全局快捷键

- [ ] 引入 `tauri-plugin-global-shortcut`
- [ ] 注册：显示/隐藏剪贴板窗（默认 Cmd/Ctrl+...）、打开偏好、纯文本粘贴、快捷粘贴
- [ ] 快捷键可在偏好设置中自定义，变更时由 Rust 重新注册
- [ ] 冲突检测与注册失败反馈

### 5.2 自启动

- [ ] 引入 `tauri-plugin-autostart`
- [ ] 自定义 `is_autostart()` 检测（eco-autostart 思路）
- [ ] 偏好设置开关联动

### 5.3 单实例

- [ ] 引入 `tauri-plugin-single-instance`，二次启动时唤起已有窗口

---

# 阶段 6 · 设置/Store 持久化（Rust 落盘）

### 6.1 设置数据模型

- [ ] Rust 定义 `Settings` 结构体（对齐旧项目 global + clipboard 两组）：
  - app：autoStart、showMenubarIcon、showTaskbarIcon、silentStart
  - appearance：theme（auto/light/dark）
  - shortcut：各快捷键
  - clipboard.content：autoPaste、copyPlain、pastePlain、deleteConfirm、操作按钮、showOriginalContent
  - clipboard.history：duration、maxCount、unit
  - clipboard.search：autoClear、defaultFocus、position
  - clipboard.window：position、style、backTop、showAll
  - update：autoUpdate、beta
- [ ] 命令：`get_settings()` / `update_settings(patch)`，落盘 JSON（app data dir）
- [ ] 默认值与缺字段兼容（旧配置升级）
- [ ] 备份机制（写入前备份 `.store-backup.json`）

### 6.2 前端绑定

- [ ] 前端 Valtio store 仅作为设置的本地镜像，变更经命令写回 Rust
- [ ] 启动时从 Rust 拉取设置初始化

---

# 阶段 7 · 前端 UI（MVP）

### 7.1 应用骨架

- [ ] `App.tsx`：HeroUI Provider + 主题 + i18n + 路由
- [ ] 全局事件监听 hook `useTauriListen`（封装 Tauri event）
- [ ] 暗色模式应用（监听系统 / 跟随设置），注入 CSS 变量

### 7.2 剪贴板历史列表（main 窗口）

- [ ] 列表容器：`react-virtuoso` 虚拟滚动
- [ ] 历史项组件：按 type 渲染（text/rtf/html/image/files 各自卡片）
- [ ] 从 Rust 命令拉取分页数据；监听「剪贴板更新」事件增量刷新
- [ ] 操作：复制回 / 粘贴 / 收藏 / 删除 / 备注（调用 Rust 命令）
- [ ] HTML 预览（DOMPurify sanitize）、Markdown 渲染、RTF 渲染、图片预览
- [ ] 选中态、键盘上下选择、Enter 粘贴、Esc 隐藏

### 7.3 搜索

- [ ] 搜索框组件（位置 top/bottom 可配）
- [ ] 输入 → 调 Rust `search_history`（FTS5）→ 渲染
- [ ] 命中文本高亮（react-mark.js 等价）
- [ ] 默认聚焦 / 自动清空（按设置）

### 7.4 偏好设置（preference 窗口）

- [ ] 设置页框架（分组：常规 / 剪贴板 / 快捷键 / 外观 / 关于）
- [ ] 各项控件绑定 Rust `get_settings`/`update_settings`
- [ ] 快捷键录制控件
- [ ] 主题切换、语言切换

### 7.5 i18n

- [ ] 引入 i18next + react-i18next
- [ ] 文案表：zh-CN（默认）/ zh-TW / en-US / ja-JP
- [ ] 语言可由 Rust 提供系统 locale 作初始值（`tauri-plugin-os`/locale）

> **MVP 里程碑**：到此可完成「监听→入库→列表展示→搜索→粘贴→设置」闭环。

---

# 阶段 8 · 增强功能（MVP 后）

### 8.1 历史清理后台任务（Rust）★

- [ ] 后台定时任务：按 `duration` + `maxCount` 清理（保留收藏项可选）
- [ ] 启动时执行一次 + 周期执行

### 8.2 分组 / 收藏视图

- [ ] 前端分组 tab、收藏过滤；Rust 查询支持 group/favorite 过滤

### 8.3 自动行为

- [ ] auto-favorite（按规则自动收藏，Rust 判定）
- [ ] auto-paste 模式（never / 双击 / 直接）
- [ ] auto-sort（时间 / 频率）

### 8.4 声音通知

- [ ] 复制成功提示音（前端播放或 Rust 触发）

### 8.5 瀑布流 / 多视图

- [ ] 瀑布流布局（react-masonry-css 等价）切换

---

# 阶段 9 · 打包、签名与发布

### 9.1 图标与资源

- [ ] 图标生成脚本（`scripts/buildIcon.ts` 等价），生成各平台格式
- [ ] 应用 identifier / name / version（version 取自 package.json）

### 9.2 自动更新

- [ ] 引入 `tauri-plugin-updater`，配置更新端点
- [ ] 配置签名密钥（`TAURI_SIGNING_PRIVATE_KEY`），前端检查更新 UI

### 9.3 CI/CD（GitHub Actions）

- [ ] release.yml：tag `v*` 触发
- [ ] 矩阵构建：macOS（arm64/x64）、Windows（x64/arm64）
- [ ] 缓存 Rust/pnpm 依赖
- [ ] 生成 changelog（changelogithub）+ 创建 Release
- [ ] macOS 签名/公证、Windows NSIS

### 9.4 版本管理

- [ ] release-it 配置（release / release-rc / release-beta）

---

# 阶段 10 · 测试与质量保障

- [ ] Rust 仓储层 / 内容识别单元测试（贯穿各阶段同步补齐）
- [ ] Tauri 命令集成测试
- [ ] 前端关键组件测试（Vitest + Testing Library）
- [ ] 跨平台手动验收清单（macOS / Windows 各核心路径）
- [ ] 性能验证：大量历史记录下的列表滚动与搜索延迟
- [ ] 内存/CPU 占用对比旧版（剪贴板监听不应空转占用）

---

# 迁移对照速查（旧 → 新）

| 旧实现                          | 新实现                        | 归属 |
| ------------------------------- | ----------------------------- | ---- |
| Kysely 前端直查 SQL             | sqlx 仓储层 + Tauri 命令      | Rust |
| 前端 `getClipboardTextSubtype`  | Rust 内容类型识别             | Rust |
| 前端轮询/回调监听剪贴板         | Rust OS 级 `ClipboardWatcher` | Rust |
| 前端窗口定位数学                | Rust 窗口定位计算             | Rust |
| 前端 store 落盘 JSON            | Rust `Settings` 落盘          | Rust |
| Ant Design 5                    | HeroUI v3                     | 前端 |
| UnoCSS                          | TailwindCSS v4                | 前端 |
| React 18                        | React 19                      | 前端 |
| Valtio（业务+UI）               | Valtio（仅 UI/设置镜像）      | 前端 |
| eco-window/paste/autostart 插件 | 重写/复用同名能力             | Rust |

---

## 推进建议

- 严格按阶段顺序：**数据层 → 监听 → 窗口 → 粘贴 → 快捷键 → 设置 → UI**，每阶段结束跑通一个可验证的小闭环。
- 每完成一个 Rust 命令，立刻在前端用最简调用验证 IPC 通路，再做 UI。
- sqlx 启用编译期校验前，先跑 `sqlx prepare` 生成离线缓存，避免 CI 无 DB 时编译失败。
- 平台特化代码（macOS/Windows）用 `#[cfg(target_os)]` 隔离，优先打通 macOS（参考项目主力平台）。
