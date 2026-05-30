# EcoPaste 重构 TODO

> 参考项目（旧版，只读参考，**不要修改**）：
>
> - 公开仓库：https://github.com/EcoPasteHub/EcoPaste
> - 本地副本（可选加速）：`/Users/ayang/Documents/PersonalProject/2024/EcoPaste_bak` — 若已 clone 到此路径，能访问文件系统的 AI 工具优先本地读取（可直接 grep/遍历目录，快于逐个远程 fetch）。
>
> 目标技术栈：**Tauri v2 + React 19 + HeroUI v3 + TailwindCSS v4**
> 核心原则：**逻辑尽量下沉 Rust，前端只做 UI 展示与交互**；仅当 Rust 不适合时才放前端。

## 技术选型（已确认）

| 维度        | 决策                   | 说明                                                                                    |
| ----------- | ---------------------- | --------------------------------------------------------------------------------------- |
| 代码组织    | **当前项目新建空分支** | `pnpm create tauri-app` 一键生成脚手架，逐功能从旧项目迁移参考                          |
| 桌面框架    | Tauri v2               | `tray-icon` / `protocol-asset` / `macos-private-api`                                    |
| IPC 层      | **tauri-awesome-rpc**  | 替代 Tauri 自带 IPC：WebSocket + JSON-RPC 2.0、命令异步执行不阻塞主线程、支持大 payload |
| 前端框架    | React 19               | 新特性：Actions、`use`、`useOptimistic`、ref as prop                                    |
| UI 组件库   | HeroUI v3              | 替换旧项目的 Ant Design 5                                                               |
| 样式        | TailwindCSS v4         | 替换旧项目的 UnoCSS；CSS-first 配置（`@theme`）                                         |
| Rust 数据层 | **sqlx (SQLite)**      | 异步 + 编译期 SQL 校验 + 内置 migration                                                 |
| 前端状态    | **Valtio**             | 仅管理 UI 状态，业务状态来自 Rust                                                       |
| 构建        | Vite + pnpm            |                                                                                         |
| Lint/Format | Biome                  |                                                                                         |
| 范围策略    | **MVP 先行**           | 阶段 0–7 为 MVP，阶段 8+ 为增强                                                         |
| 支持平台    | **仅 macOS + Windows** | 不支持 Linux；平台特化用 `#[cfg(target_os)]` 隔离                                       |

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

- [x] 规划 `src-tauri/src` 模块目录：`commands/` `db/` `clipboard/` `window/` `paste/` `settings/` `core/`（platform setup）
- [x] `lib.rs` 搭建 Tauri `Builder` 骨架，`main.rs` 调用 `lib::run()`
- [x] 引入 `serde` / `serde_json` / `thiserror` / `anyhow`（错误处理）
- [x] 定义统一错误类型 `AppError` + `Result<T>`，并实现 `serde::Serialize` 供命令返回
- [x] 引入 `tracing` 或 `tauri-plugin-log` 做日志
- [x] 接入 [`tauri-awesome-rpc`](https://github.com/ahkohd/tauri-awesome-rpc) 替代 Tauri 自带 IPC
  - [x] 后端：`AwesomeRpc::new(allowed_origins)` + `Builder::invoke_system(init_script)`，`setup` 中启动 WebSocket
  - [x] 前端：安装 `tauri-awesome-rpc` npm 包，`invoke` / `listen` / `once` 透明切换至 WS 通道
  - [ ] 统一规范 `EmitterExt` / `ListenerExt` 用法（避免与 `tauri::Emitter` 的 trait 冲突）
  - [x] 调优 max payload / buffer / max connections，预留大文件与图片 base64 等大 payload 场景
  - [x] 评估 dev / 打包后的 origin 白名单与端口策略

---

# 阶段 1 · 数据层（Rust + sqlx）★ 核心

### 1.1 数据库初始化

- [x] 引入 `sqlx`（features：`runtime-tokio` / `sqlite` / `macros` / `chrono`）
- [x] 确定 DB 路径：app data dir 下 `clipboard.db`（dev 用 `clipboard.dev.db`），由 Rust 解析路径
- [x] 实现 `db::init()`：创建/打开数据库 + `SqlitePool` 连接池，存入 Tauri `State`
- [x] 启用 SQLite pragma（`journal_mode=WAL`、`foreign_keys=ON`、`synchronous=NORMAL`）

### 1.2 Migration（编译期校验）

- [x] 建立 `migrations/` 目录，配置 `sqlx::migrate!`
- [x] 编写 `0001_init.sql`：`clipboard_items` 表 + `clipboard_groups` 表
  - `clipboard_items` 字段：`id TEXT PK`、`kind`、`sub_kind`、`group_id`、`content`、`search_text`、`size INTEGER`、`width`、`height`、`use_count INTEGER`、`is_favorite INTEGER`、`is_pinned INTEGER`、`platform`、`note`、`created_at`、`updated_at`
  - `clipboard_groups` 字段：`id TEXT PK`、`name`、`sort_order INTEGER`、`created_at`；`clipboard_items.group_id` FK 到此表，`ON DELETE SET NULL`
  - 类型枚举：`kind`：`text` / `image` / `files`；`sub_kind`：`rtf` / `html` / `url` / `email` / `color` / `path`
- [x] 编写 `0002_fts.sql`：FTS5 虚表 `clipboard_items_fts` + 触发器（insert/update/delete 同步）
- [x] 启动时自动执行 migration，失败时优雅报错
- [x] 配置 `.env`（`DATABASE_URL`）；仓储层最终改用运行时 `query_as` + `QueryBuilder`（见 1.3），未启用 `query!` 宏的编译期校验，故无需 `sqlx prepare` 生成 `.sqlx`。SQL 正确性由仓储层单测兜底

### 1.3 数据模型与仓储层

- [x] 定义 Rust 结构体 `ClipboardItem` / `ClipboardGroup`（`#[derive(Serialize, Deserialize, FromRow)]`）
- [x] 定义查询参数结构体（分页、过滤 kind、favorite、pinned、group、关键词；排序按时间/频率）
- [x] 实现 `clipboard_items` 仓储函数（`src-tauri/src/db/items.rs`，全部 async，统一 `Result<T, AppError>`，首参 `pool: &SqlitePool`）：
  > 用运行时 `sqlx::query_as::<_, ClipboardItem>` + `QueryBuilder`（复用模型的 `FromRow`/`sqlx::Type` derive），不用 `query_as!` 宏——动态过滤/分页 SQL 无法用静态 SQL 宏表达，且免去编译期连库与 `.sqlx` 离线缓存维护。
  > 方法名一律带 `_item` / `_items` 后缀（与 `clipboard_groups` 的 `*_group` 后缀对称），即便日后从 `db::items` / `db::groups` 扁平化 re-export 也不会撞名。
  - [x] `insert_item(item: &ClipboardItem) -> Result<()>`
  - [x] `query_items(q: &ClipboardItemQuery) -> Result<Vec<ClipboardItem>>` — 统一列表查询：按 `kind` / `group_id` / `favorite` / `pinned` 过滤 + `limit`/`offset` 分页 + `sort`（`CreatedAtDesc` / `UseCountDesc`，置顶项恒前置）；`keyword` 非空时委托 `search_items_fts`
  - [x] `search_items_fts(q: &ClipboardItemQuery) -> Result<Vec<ClipboardItem>>` — 基于 `clipboard_items_fts` 的关键词前缀检索（由 `query_items` 在有 `keyword` 时调用，复用同一套过滤/分页/排序；私有 `fts_match_expr` 把关键词转成转义后的 `"x"*` 前缀表达式）
  - [x] `find_item_by_id(id: &str) -> Result<Option<ClipboardItem>>`
  - [x] `toggle_item_favorite(id: &str)` / `toggle_item_pinned(id: &str)` — 翻转 `is_favorite` / `is_pinned`
  - [x] `update_item_note(id: &str, note: Option<&str>) -> Result<()>`
  - [x] `increment_item_use_count(id: &str) -> Result<()>` — `use_count + 1` 并刷新 `updated_at`
  - [x] `delete_item(id: &str) -> Result<()>` / `delete_items(ids: &[String]) -> Result<u64>` / `clear_items(keep_favorite: bool) -> Result<u64>`（后两者返回删除行数）
- [x] 实现 `clipboard_groups` 仓储函数（`src-tauri/src/db/groups.rs`，全部 async，统一 `Result<T, AppError>`，首参 `pool: &SqlitePool`；分组特性见阶段 8.2，数据层先备齐基础 CRUD）：
  - [x] `list_groups() -> Result<Vec<ClipboardGroup>>` — 按 `sort_order` 升序（同序按 `created_at` 兜底，顺序稳定）
  - [x] `insert_group(group: &ClipboardGroup) -> Result<()>` / `rename_group(id: &str, name: &str) -> Result<()>` / `delete_group(id: &str) -> Result<()>`（删除分组时其下记录的 `group_id` 由 FK `ON DELETE SET NULL` 自动置空）
- [x] 写仓储层单元测试（用内存库 `sqlite::memory:` + `sqlx::migrate!` 跑迁移）
  > 共用 `db::test_support::memory_pool()`（`#[cfg(test)]`，单连接 + 开外键 + 跑迁移）；`items` 覆盖 增删改查 / 过滤 / 排序（置顶前置）/ 分页 / FTS 前缀检索 / 计数；`groups` 覆盖 CRUD / 排序 / FK `ON DELETE SET NULL`。

### 1.4 去重与计数逻辑（Rust）

- [x] 入库前对相同内容做比对，命中已有记录则 `use_count + 1` 并刷新 `updated_at`，不再插入新行（替代旧前端逻辑）
  > 实现为 `db::items::upsert_item`：按 `content_hash` 查重，命中走 `increment_item_use_count` 并返回 `UpsertResult { id, deduplicated }`，未命中走 `insert_item`。
- [x] 选择比对策略（text 按 `content` 直接比对；image/files 用内容哈希）；如需按哈希查重，评估在 `clipboard_items` 增加 `content_hash` 列并建索引（写新 migration `0003_*.sql`，迁移只增不改）
  > 统一用 `content_hash = sha256("<kind>:<content>")`（`db::items::content_hash`）一条索引路径覆盖两种策略：text 即哈希内容串；image/files 的 `content` 为落盘引用/路径，阶段 2.3 持有原始字节时可改为对字节哈希后写入。`kind` 前缀隔离避免跨类型误判。
  > **未发版、仍在开发阶段**，故直接改 `0001_init.sql` 增列 `content_hash TEXT NOT NULL` + 索引 `idx_clipboard_items_content_hash`，不另写 `0003`（迁移只增不改的约定从首个发版后生效）。

---

# 阶段 2 · 剪贴板监听与内容处理（Rust）★ 核心

### 2.1 剪贴板读取插件

- [x] 评估 `tauri-plugin-clipboard-x` 复用 vs 自研 → 选 **底层库 `clipboard-rs`**（ChurchTao/clipboard-rs `0.3`）：text/html/rtf/image/files 全支持、macOS/Windows 原生读取、内置 watcher（供 2.2）、`RustImage` 缩略图（供 2.3）、`set_*` 写回（供 4.1）；比再包一层插件更可控
- [x] 引入剪贴板能力，支持读取：纯文本 / HTML / RTF / 图片 / 文件列表
- [x] 封装 `ClipboardReader::read_all() -> Result<Option<ClipboardPayload>>`（带类型标记，files > image > text 优先级；`None` = 空/无可识别内容）
  > `src-tauri/src/clipboard/{payload,read}.rs`。`ClipboardPayload` 为内部类型（不直接回传前端）；`get_files()` 返回裸路径（macOS 走 `NSURL.path()` 已解码），无需 URI 解码。子类型识别/图片落盘留待 2.3。真实剪贴板往返读取测试见 `read.rs`（`#[ignore]`，`cargo test -- --ignored` 验证，已通过）。

### 2.2 OS 级监听（替代前端轮询）★

- [x] macOS：基于 `NSPasteboard` 的 `changeCount` 轮询线程（或事件）→ 由 `clipboard-rs` watcher 内置
- [x] Windows：`AddClipboardFormatListener` 监听 `WM_CLIPBOARDUPDATE` → 由 `clipboard-rs` watcher 内置
- [x] 统一抽象 `ClipboardWatcher`，变更时触发回调
  > `src-tauri/src/clipboard/watcher.rs`：`ClipboardChangeHandler` 实现 `clipboard_rs::ClipboardHandler`，在名为 `clipboard-watcher` 的独立 `std::thread` 上跑阻塞 `start_watch()`。平台句柄（`ClipboardContext`/`ClipboardWatcherContext`）**在该线程内构造**，绕开 `!Send` 约束。
- [x] 监听线程产出变更 → 经内容处理 → 入库 → `emit` 事件通知前端刷新
  > 回调里同步 `read_all()` → `ingest::build_item()` 转换（含图片落盘）→ `tauri::async_runtime::spawn` 投递（仅移动 `Send` 的 pool/AppHandle/item）→ `db::items::upsert_item` 去重入库 → `EmitterExt::emit("clipboard://updated", {id, deduplicated})`。text/files/image 全部接通。
- [x] 处理自身写回剪贴板导致的回环（写回时打标记/抑制下一次监听）
  > `src-tauri/src/clipboard/guard.rs`：`WritebackGuard`（入 Tauri `State`，供 4.1 写回前 `suppress(content_hash)`）。回调用 `should_skip(hash)` 按指纹比对命中即跳过，带 2s TTL 兜底防登记滞留误吞后续真实复制。
  > 验证：`guard.rs` 单测覆盖抑制/不误伤/过期；`watcher.rs` 含 `#[ignore]` 真实剪贴板端到端测试（读取→转换→去重入库 + 写回抑制），触碰系统剪贴板的测试经 `clipboard::test_lock` 串行化，默认多线程 runner 下亦稳定。`cargo test -- --ignored` 已通过。

### 2.3 内容类型识别（Rust）★

- [x] 文本子类型检测：`url` / `email` / `color`（hex/rgb(a)/hsl(a)）/ `path`
  > `src-tauri/src/clipboard/detect.rs`：纯函数 `detect_text_sub_kind`，判定顺序 url > email > color > path（对齐旧项目）。正则用 `std::sync::LazyLock` 编译期复用。两处比旧版收紧以求稳：url 要求带协议头或 `www.`（不再把任意带点词判为链接）；path 仅认**绝对路径 + `exists()`**（相对路径受进程 cwd 影响，监听线程下不可控）。color 只匹配 hex/函数式语法，天然排除 CSS 关键字，无需旧版的关键字黑名单。
- [x] 生成 `search` 字段：HTML→纯文本、RTF→纯文本、image→占位、files→路径串
  > 存储语义（`ingest.rs`，对齐参考插件 [tauri-plugin-clipboard-x](https://github.com/ayangweb/tauri-plugin-clipboard-x) 读取优先级 html > rtf > plain）：`content` 存源表示（`get_html()`/`get_rich_text()` 原文或纯文本），`search_text` 存纯文本供 FTS。**HTML/RTF 不自己解析转纯文本**——复制富文本时剪贴板本就并存 `get_text()` 纯文本表示，直接拿来当 `search_text` 即可（前端再用 DOMPurify 渲染 HTML 源）。image 的 `search_text` 留 `None`（不污染 FTS，占位展示交前端按 `kind` 渲染）；files 的 `content` 即换行路径串、本身可检索。
- [x] 图片处理：解码取 `width`/`height`，落盘到 app data，`content` 存引用
  > `src-tauri/src/clipboard/storage.rs`：`ImageStore`（入 Tauri `State`）。**按内容哈希分片**避免单目录爆量：`<app_data>/resources/images/{origin,thumbnails}/<sha256前2位>/<sha256>.png`。`content` 存文件名 `<sha256>.png`，分片目录由 `origin_path`/`thumbnail_path`（供阶段 4 写回与前端预览）从文件名推导。缩略图最长边 300px。文件名取 PNG 字节的 sha256 → 同图重复复制落盘幂等，且 `content_hash` 对字节敏感，与 1.4 去重同源。
- [x] 文件处理：读取路径列表（裸路径，2.1 已确认无需 URI 解码）；`content` 存换行连接路径串
  > 单文件大小/MIME 等逐文件元信息留待 UI 需要时再补（前端展示阶段 7.2），数据层已可入库检索。
- [x] 单元测试覆盖各类型识别
  > `detect.rs` 覆盖 url/email/color/path 正负例；`storage.rs` 覆盖哈希分片落盘/幂等/路径解析（合成 PNG）；`ingest.rs` 覆盖 html/rtf/plain 存储语义（富文本用 OS 纯文本作检索文本）+ 图片落盘记录 + 空文本跳过；`watcher.rs` 新增 `#[ignore]` 真实剪贴板图片端到端（OS TIFF→PNG 解码→落盘→入库）。`cargo test`（40 通过）+ `cargo test -- --ignored`（4 通过）+ `cargo fmt --check` + clippy 均过。

### 2.4 暴露命令

- [x] `#[tauri::command]` 包装 2.1–2.3，供前端按需手动触发（如「重新读取」）
  > `src-tauri/src/commands/clipboard.rs`（薄封装层，仅参数校验 + 调下层）：
  >
  > - `read_clipboard()` — 手动重新读取并入库，复用监听管线 read_all → `build_item` → `persist_and_notify`（去重入库 + emit `clipboard://updated`），与 OS 监听同一条路径、语义一致。返回 `Option<ReadClipboardResult { item, deduplicated, captured }>`，剪贴板空时为 `None`。
  > - `get_clipboard_image_path(file_name, thumbnail)` — 把图片文件名解析为原图/缩略图绝对路径，供前端预览取图。
  >   关键点：(1) `ClipboardReader` 的 `ClipboardContext` 是 `!Send`，读取+转换全在 await 前的同步块内完成并 drop，命令 future 才满足 `Send`；(2) `read_clipboard` 与 watcher 共用抽出的 `clipboard::persist_and_notify`，不重复入库/emit 逻辑；(3) `get_clipboard_image_path` 的 `file_name` 是唯一外部输入，`validate_image_file_name` 防路径穿越（拒绝分隔符/`..`/子目录/非 `.png`），单测覆盖。
  >   awesome-rpc 只换 IPC 传输层，命令仍走标准 `generate_handler!`（`lib.rs` 注册）。`cargo test`（42 通过）+ clippy + fmt 均过。

> **阶段 2 完成**：剪贴板「监听 → 读取 → 内容识别/图片落盘 → 去重入库 → emit」闭环已通，并提供手动重读 / 取图命令。

---

# 阶段 3 · 窗口管理（Rust + 平台特化）★ 核心

### 3.1 窗口配置

- [x] `tauri.conf.json` 定义两个窗口：
  - `main`：360×600、透明、无边框、置顶、跳过任务栏、默认隐藏、`#/`
  - `preference`：700×480、居中、可调、`#/preference`
    > 两窗口均 `visible: false`（启动隐藏，由 3.2/3.3 的 show/hide 命令控制显隐）。`main` 加 `resizable:false` + `maximizable:false`（固定浮层尺寸）；`preference` 设 `minWidth/minHeight=700×480` 防缩到不可用。透明窗口依赖 macOS 私有 API：`tauri.conf.json` 开 `app.macOSPrivateApi:true`，`Cargo.toml` 的 `tauri` 同步加 `macos-private-api` feature（否则编译报错）。URL 用 `index.html/#/` 哈希路由，对齐前端 `createHashRouter`。NSPanel 浮层级 / 窗口特效（sidebar）等留待 3.3。
- [x] 配置 capabilities/permissions（v2 权限模型，最小授权）
  > [capabilities/default.json](src-tauri/capabilities/default.json)：`windows` 从 `["main"]` 扩到 `["main","preference"]`（新 preference 窗口纳入授权，否则收不到 core 事件）。权限收敛到仅 `core:default`（含 `core:window`/`core:event`/`core:app` 等基础能力，足够前端监听 emit 事件与基本窗口操作）。自定义命令（`read_clipboard` 等）经 `generate_handler!` 注册、属应用自有命令，**不受 ACL 约束**，无需在此授权；awesome-rpc 仅换 IPC 传输层不影响这一点。后续插件能力（window 管理 / paste / 快捷键 / 设置落盘）按阶段最小增量添加。
  > 顺手清理脚手架残留：`opener:default` 权限连同从未注册/引用的 `tauri-plugin-opener` 依赖一并移除（Cargo.toml + Cargo.lock），保持最小授权与零死依赖。`cargo check` 通过。

### 3.2 自定义窗口插件（eco-window 思路）

- [x] 命令：`show_window` / `hide_window` / `toggle_window` / `show_taskbar_icon`
  > 不做成插件，直接作为 `window/mod.rs` 的公共函数 + `commands/window.rs` 的 `#[tauri::command]` 薄封装。`show_window` 执行 show → unminimize → set_focus；`hide_window` 仅 hide；`toggle_window` 按 `is_visible` 分派；`show_taskbar_icon` 平台分支：macOS 调 `set_dock_visibility`，Windows 调 `set_skip_taskbar(!visible)`。`window/mod.rs` 导出 `MAIN_WINDOW_LABEL` / `PREFERENCE_WINDOW_LABEL` 常量供后续模块使用。NSPanel 浮层级特化留待 3.3 接入 `tauri-nspanel` 后在 `show_window`/`hide_window` 中对 main 窗口做分支处理。
- [x] 窗口定位计算（Rust）★：跟随光标 / 屏幕居中 / dock 底部；多显示器下取光标所在屏
  > `window/position.rs`：定义 `WindowPosition`（`Remember`/`Follow`/`Center`）和 `WindowStyle`（`Standard`/`Dock`）枚举（`serde` camelCase，阶段 6 设置可直接复用）。核心函数 `position_window(window, style, position)`：先 `monitor_from_cursor` 遍历 `available_monitors()` 命中光标所在屏（物理坐标比对），无命中则 fallback 首个显示器；再按模式分派——`Follow` 将窗口左上角置于光标处并 clamp 到屏幕右/下边界、`Center` 居中于光标所在屏、`Dock` 全宽 400px 贴底（修复旧版多显示器 y 偏移 bug：加上 `monitor.position.y`）、`Remember` 不动。`window/mod.rs` 代理 `position_window(app, label, style, pos)` 取窗口再调内层。命令层 `commands/window.rs` 暴露 `position_window` 供前端调用。`cargo check` 通过。
- [x] 窗口位置/尺寸持久化（由 Rust 落盘 `.window-state.json`），启动恢复
  > `window/state.rs`：`WindowStateStore` 持有文件路径 + `Mutex<HashMap<String, WindowState>>`，入 Tauri `State`，`setup` 中初始化。文件名区分环境：dev `window-state.dev.json` / prod `window-state.json`，存放于 `app_local_data_dir` 根目录（对齐 DB 的 dev/prod 命名惯例）。`WindowState { x, y, width, height }` 按窗口 label 索引。`save` 写内存 + 立即落盘 JSON；`get` 仅读内存。`save_window_state(app, label)` 读取窗口当前 `outer_position` + `inner_size` 后存储；`restore_window_state(app, label)` 从存储恢复位置和尺寸，返回 `bool` 表示是否有状态可恢复。命令层 `commands/window.rs` 暴露 `save_window_state` / `restore_window_state` 供前端在窗口 move/resize 事件和启动时调用。`cargo check` 通过。

### 3.3 macOS NSPanel 特化（暂时不实现，在等 tauri 的新特性）

- [ ] 引入 `tauri-nspanel`，将 main 窗口转为 NSPanel（浮层、dock level）
- [ ] 隐藏 dock 图标、可在全空间显示、`acceptsFirstMouse`
- [ ] 绑定 focus/blur/resize/move 事件 → emit 给前端
- [ ] 失焦自动隐藏（可配置）

### 3.4 关闭行为

- [x] `WindowEvent::CloseRequested`：隐藏而非退出（常驻后台）
- [x] macOS 应用 reopen 事件处理
  > 逻辑下沉 `window/mod.rs`，`lib.rs` 只做 Builder 接线：`hide_on_close(window)` 隐藏窗口并返回 `true`，`.on_window_event` 里据此 `api.prevent_close()`（任一窗口关闭都隐藏，常驻托盘后台）。reopen 需 build-then-run 形态，故 `.run(generate_context!())` 改为 `.build(...).run(|app_handle, event| ...)`：macOS 下匹配 `RunEvent::Reopen`，`has_visible_windows` 为真直接返回，否则 `handle_reopen` 唤起 preference 窗口（对齐旧版点击 dock 图标行为）。`handle_reopen` 用 `#[cfg(target_os = "macos")]` 隔离。`cargo check` / `cargo fmt --check` / clippy 均过。

---

# 阶段 4 · 粘贴与写回剪贴板（Rust）★ 核心

### 4.1 写回剪贴板

- [x] `write_to_clipboard(item)`：按类型写回 text/rtf/html/image/files
- [x] 纯文本模式：剥离格式只写 plain text
  > `clipboard/write.rs`：`write_to_clipboard(store, guard, item, plain)` 内部新建 `ClipboardContext` 后按 `kind` 分派——Text 走 `set_html` / `set_rich_text` / `set_text`（依 `sub_kind`，未识别子类型与 url/email/color/path 均走纯文本通道）；Image 从 `ImageStore::origin_path` 读原 PNG → `RustImageData::from_bytes` → `set_image`；Files 把 `content` 按 `\n` 拆回路径列表 → `set_files`。写回前一律向 `WritebackGuard` 登记将写入内容的 `content_hash`，与 watcher 路径上 `build_item` 算出的哈希同源，OS 监听到自身写回时直接 `should_skip` 跳过入库，杜绝「点击粘贴 → 自动新增一条」回环。`plain = true` 忽略 `sub_kind`，优先写 `search_text`（即 OS 提供的纯文本表示），缺失时退回 `content`。
  > 命令层 `commands/clipboard.rs::write_to_clipboard(id, plain)`：按 id 查记录后调下层（同 `read_clipboard` 写法，`!Send` 的 `ClipboardContext` 在同步段内创建与销毁，命令 future 仍 `Send`）。已在 `lib.rs` 的 `generate_handler!` 注册。`cargo check` 通过；`clipboard::write` 三个 `--ignored` 真实剪贴板测试全过（plain 写回 + guard 抑制、`plain` 模式剥离 HTML、Image 往返哈希一致并被 guard 抑制——确认 OS pasteboard PNG 字节往返无损，文档里的「极端情况漏抑制」未在测试中复现）。

### 4.2 模拟粘贴（eco-paste 思路）

- [x] macOS：`core-graphics` (CGEvent) 触发 ⌘V
  > `keystroke/macos.rs::simulate_paste()` 走 `CGEventSource::CombinedSessionState` + `CGEvent::new_keyboard_event(KEY_V=0x09, down/up)`，设 `CGEventFlagCommand` 后 `post(CGEventTapLocation::HID)`。比旧版 `osascript` 实现少一层子进程，但同样需要「辅助功能」权限（CGEvent 未授权时被静默丢弃，是 macOS 安全模型限制）。
- [x] Windows：`winapi` SendInput 模拟 Shift+Insert
  > `keystroke/windows.rs::simulate_paste()` 一次性投递 4 个 `INPUT_KEYBOARD`（Shift↓ / Insert↓ / Insert↑ / Shift↑）。选 Shift+Insert 而非 Ctrl+V 是因为前者是 NT 时代沿用的系统级编辑约定，传统 Win32 控件、cmd/PowerShell 终端、部分 Electron 应用都接收；Ctrl+V 经常被自定义快捷键吞掉或在控制台不响应。winapi 直调比 enigo 少一层依赖。
- [x] `paste_clipboard_item(id, plain)` 组合命令：写回 + 隐藏窗口 + 触发粘贴
  > `commands/clipboard.rs::paste_clipboard_item`：复用 `find_item_by_id` + `clipboard::write_to_clipboard`，结尾 `keystroke::simulate_paste()`。命名从 `paste` 改为 `paste_clipboard_item`，避免与前端 HTML paste 事件等通用术语撞名，并与 `read_clipboard` / `write_to_clipboard` 命名风格对齐。注：Rust 模块经历两轮改名 `paste/` → `simulate_paste/` → `keystroke/`，最终落在「按键事件注入」这个领域名词上，调用点 `crate::keystroke::simulate_paste()` 不再有词根重复；后续若有其他系统级按键注入（如快捷键测试）也能收纳进来。写回剪贴板早在 4.1 下沉到 `clipboard/write.rs`。
- [x] 粘贴前隐藏主窗口的时序处理
  > `paste_clipboard_item` 内：写回后 `hide_window(main)` → `tokio::sleep(50ms)` → `simulate_paste`。50ms 是经验值（旧版 100ms 偏保守），让 OS 完成「key window 交还上一个应用」的焦点切换；过短按键会在自家窗口仍是 key window 时被吞掉，过长用户能感到延迟。隐藏失败仅 `log::warn!` 不中断（剪贴板已写回，按键仍可生效）。**不做**主动「聚焦回原应用」——按用户指示，后续会把主窗口改成「不抢占焦点」（NSPanel `NonactivatingPanel` / Windows `WS_EX_NOACTIVATE`），届时 hide + sleep 整段可拆除；现在的实现是过渡形态，避免重复堆 OS 级前台窗口观察器。新增 `tokio` 主依赖 `time` feature（原先只在 dev-deps）。

---

# 阶段 5 · 全局快捷键与自启动（Rust）

### 5.1 全局快捷键

- [x] 引入 `tauri-plugin-global-shortcut`
- [x] 注册：显示/隐藏剪贴板窗（默认 Cmd/Ctrl+...）、打开偏好、纯文本粘贴、快捷粘贴
- [x] 快捷键可在偏好设置中自定义，变更时由 Rust 重新注册
- [x] 冲突检测与注册失败反馈
  > 新增 `src-tauri/src/shortcut/mod.rs`：`ShortcutManager`（`Mutex<bindings + active>` 作为 Tauri State）+ `init/apply/current_bindings` 三个入口。`apply` 全量替换：先 unregister 上一轮已注册项再逐个 register 新项，单项失败不影响其它项——`log::warn!` 落盘 + `app.emit("shortcut://conflict", {action, binding, reason})` 通知前端（让偏好页能高亮冲突格子并提示用户改键）。`register_one` 先 `is_registered` → `unregister` 兜底其它进程占用残留，再 `on_shortcut` 注册回调；回调内只在 `ShortcutState::Pressed` 触发，否则按下/松开会让 toggle 一来一回回到原态。默认绑定对齐旧版 `stores/global.ts`：`open_clipboard=Alt+C` → `toggle_window("main")`，`open_preference=Alt+X` → `toggle_window("preference")`。空字符串视为禁用直接跳过，给后续「不绑定该项」留口子。
  > 命令层 `commands/shortcut.rs::{get_shortcuts, update_shortcuts}`：前端读当前绑定 / 偏好页变更后回写。**重注册机制现在就位**（命令 + apply 全量替换），但默认值仍是硬编码；待阶段 6 设置系统落盘后，启动时改为从 Settings 加载初始绑定、偏好页变更走 `update_settings → apply`。**未注册：`pastePlain` 和 `quickPaste`**——前者在旧版是 `useKeyPress` 局部按键（焦点在主窗时才生效），后者依赖列表序号上下文，都是「窗口内」交互，留到阶段 7 前端用 `useKeyPress` 实现。`lib.rs`：注册 `tauri_plugin_global_shortcut::Builder::new().build()`，setup 末尾调 `shortcut::init(&handle)`（必须在 plugin 注册之后，因 `app.global_shortcut()` 依赖 plugin state）；旧 `block_on` 闭包 move 了 handle，故拆出 `handle_db` 副本喂给 async 段，原 `handle` 留给 `shortcut::init`。Cargo 新增 `tauri-plugin-global-shortcut = "2.3"`。capabilities 不动——全部注册在 Rust 侧，前端不需要 `global-shortcut:allow-*` 权限。`cargo check` / `cargo fmt` / `cargo clippy --lib`（仅看新增模块）均过；db/models.rs 有的 22 条 dead_code 警告是阶段 2 留下的（FTS5、查询模型尚无调用方），与本改动无关。

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
- [ ] 输入 → 调 Rust 列表查询命令（`ClipboardItemQuery.keyword` 带关键词时自动走 FTS5）→ 渲染
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
