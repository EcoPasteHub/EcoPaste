# EcoPaste 重构 TODO

> 参考项目（旧版，只读参考，**不要修改**）：
>
> - 公开仓库：https://github.com/EcoPasteHub/EcoPaste
> - 本地副本（可选加速）：`/Users/ayang/Documents/PersonalProject/2024/EcoPaste_bak` — 若已 clone 到此路径，能访问文件系统的 AI 工具优先本地读取（可直接 grep/遍历目录，快于逐个远程 fetch）。
>
> 目标技术栈：**Tauri v2 + React 19 + HeroUI v3 + TailwindCSS v4**
> 核心原则：**逻辑尽量下沉 Rust，前端只做 UI 展示与交互**；仅当 Rust 不适合时才放前端。

## 技术选型（已确认）

| 维度        | 决策                   | 说明                                                                                                                                       |
| ----------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 代码组织    | **当前项目新建空分支** | `pnpm create tauri-app` 一键生成脚手架，逐功能从旧项目迁移参考                                                                             |
| 桌面框架    | Tauri v2               | `tray-icon` / `protocol-asset` / `macos-private-api`                                                                                       |
| IPC 层      | **Tauri 原生 IPC**     | 原 `tauri-awesome-rpc` 在 Windows 触发 `miow 0.2.2` null 指针 panic（不可修复，上游已停 issue），改回原生 `#[tauri::command]` + `app.emit` |
| 前端框架    | React 19               | 新特性：Actions、`use`、`useOptimistic`、ref as prop                                                                                       |
| UI 组件库   | HeroUI v3              | 替换旧项目的 Ant Design 5                                                                                                                  |
| 样式        | TailwindCSS v4         | 替换旧项目的 UnoCSS；CSS-first 配置（`@theme`）                                                                                            |
| Rust 数据层 | **sqlx (SQLite)**      | 异步 + 编译期 SQL 校验 + 内置 migration                                                                                                    |
| 前端状态    | **Valtio**             | 仅管理 UI 状态，业务状态来自 Rust                                                                                                          |
| 构建        | Vite + pnpm            |                                                                                                                                            |
| Lint/Format | Biome                  |                                                                                                                                            |
| 范围策略    | **MVP 先行**           | 阶段 0–7 为 MVP，阶段 8+ 为增强                                                                                                            |
| 支持平台    | **仅 macOS + Windows** | 不支持 Linux；平台特化用 `#[cfg(target_os)]` 隔离                                                                                          |

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

### 3.3 macOS NSPanel 特化

- [x] 引入 `tauri-nspanel`，将 main 窗口转为 NSPanel（浮层、dock level）
- [x] 隐藏 dock 图标、可在全空间显示、`acceptsFirstMouse`
- [x] 绑定 focus/blur/resize/move 事件 → emit 给前端
  > `window/macos.rs`：依赖 `tauri-nspanel` v2.1 分支。`register_plugin` 在 setup 早期注册 plugin（必须在 to_panel 前）。`setup_main` 用 `tauri_panel!` 宏定义 `MainPanel`（`is_floating_panel + can_become_key_window` 但 `can_become_main_window=false`），`to_panel::<MainPanel>` 转换后设 `PanelLevel::Dock` + `StyleMask::empty().resizable().nonactivating_panel()`（NonactivatingPanel = 按键不激活 App，Spotlight/Raycast 模式），`CollectionBehavior::stationary().move_to_active_space().full_screen_auxiliary()`。注册 `MainPanelEventHandler` 把 `window_did_become_key/resign_key/resize/move` 转 emit 成 `tauri://focus|blur|move|resize` 给前端。`show_main_panel`/`hide_main_panel` 都在 `run_on_main_thread` 内执行，shown 切 `can_join_all_spaces`、hidden 切回 `move_to_active_space`，避免全空间常驻浪费。`acceptsFirstMouse` 暂走默认（点击窗外才隐藏的需求未到，后续若要"点透"再 objc2 子类化）。

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

- [x] ~~引入 `tauri-plugin-autostart`~~ 改为直接调 `auto-launch = "0.6"` crate（上游 plugin bug：tauri-apps/plugins-workspace#1922）
- [x] 自定义 `is_autostart()` 检测（eco-autostart 思路）
- [x] 偏好设置开关联动（命令已就位，UI 待阶段 7）
  > 新增 `src-tauri/src/autostart/mod.rs`：在 setup 末尾 `init(&handle)`，构造 `AutoLaunch{app_name=package_info().name, app_path=current_exe(), args=["--auto-launch"]}` 存入 `State<AutostartManager>`。命令 `commands/autostart.rs::{get_autostart, set_autostart, is_launched_via_autostart}`。`is_launched_via_autostart` 复用旧版 `eco-autostart::is_autostart` 思路——扫 `env::args()` 找 `--auto-launch` flag，给将来「静默启动」用。**不引入 `tauri-plugin-autostart`**：参考 tauri-apps/plugins-workspace#1922，macOS 上 `is_enabled` 会因为 LaunchAgent plist 路径解析差异而误报，`enable` 写入的可执行路径也可能指向 `.app` 内部而非 wrapper，造成开关之后无法卸载；`auto-launch` 直接走 `~/Library/LaunchAgents/<bundle>.plist` 并把 exe 路径作为 `ProgramArguments[0]`，绕过这个 bug。Cargo 新增 `auto-launch = "0.6"`（0.5 不支持以管理员权限运行的应用自动重启，见 issue#1922 评论 3858495164）。

### 5.3 单实例

- [x] 引入 `tauri-plugin-single-instance`，二次启动时唤起已有窗口
  > `lib.rs`：`tauri_plugin_single_instance::init` 必须作为**第一个**注册的 plugin（plugin 文档明确要求，靠前注册才能在其它 plugin 初始化前拒绝重复进程，避免数据库 / 全局快捷键 / LaunchAgent 在第二实例里被重复初始化导致状态错乱）。二次启动回调里调 `show_window(PREFERENCE_WINDOW_LABEL)`——和 macOS dock reopen 行为对齐：main 窗口是"剪贴板弹层"（跟随光标、用快捷键唤起），用户从启动器再次点图标的语义是"打开应用主面板"，应映射到偏好窗。不传 `argv`/`cwd`：本应用没有 CLI 入参语义（`--auto-launch` 只在首次启动判断），第二实例的参数没有消费场景。Cargo 新增 `tauri-plugin-single-instance = "2"`。**不需要**前端 capability 权限：plugin 完全工作在 Rust 侧、无 JS API。

---

# 阶段 6 · 设置/Store 持久化（Rust 落盘）

### 6.1 设置数据模型

- [x] Rust 定义 `Settings` 结构体（重新组织，不照搬旧版命名）：
  - general：autoStart、silentStart、trayIcon、dockIcon
  - appearance：theme（auto/light/dark）、language（zh-CN/en-US）
  - shortcuts：openClipboard、openPreference、pastePlain、quickPaste{enabled,modifier}
  - clipboard.content：autoPaste(disabled/single/double)、copyPlain、pastePlain、showOriginalPreview、deleteConfirm、autoFavorite、autoSortByFrequency、itemActions
  - clipboard.history：retention{value,unit(hours/days/weeks/months/forever)}、maxCount(0=∞)
  - clipboard.search：position、defaultFocus、clearOnHide
  - clipboard.window：style(standard/dock)、position(remember/followCursor/center)、alwaysOnTop、allWorkspaces
  - clipboard.feedback：copySound
  - update：autoCheck、includeBeta
- [x] 命令：`get_settings()` / `update_settings(patch)`，落盘 JSON（app local data dir）
- [x] 默认值与缺字段兼容（每个结构 `#[serde(default)]`，新增字段不破坏旧文件；本项目不做旧版本数据迁移）
- [x] 备份机制（写入前 rename 当前文件为 `.bak`，新内容先写 `.tmp` 再 rename，原子覆盖）
  > 新增 `src-tauri/src/settings/{model.rs,store.rs,mod.rs}` + `commands/settings.rs`。
  > **重要：不照搬旧版字段。** 旧版 `backTop` → `alwaysOnTop`、`showAll` → `allWorkspaces`、`history.{duration,unit:number}` 改成强类型 `retention{value, unit:RetentionUnit枚举}`、`autoPaste:"single"|"double"` 增加 `disabled` 档、`operationButtons` → `itemActions`（行为命名）、`audio.copy:bool` → `feedback.copySound`、`update.auto/beta` → `update.{autoCheck,includeBeta}`、`shortcut.quickPaste.value` → `quickPaste.modifier`（明确语义：按住的修饰键）。`General` 把菜单栏/任务栏图标命名为 `trayIcon`/`dockIcon`（跨平台中性词），不再用 `showMenubarIcon`/`showTaskbarIcon` 这种带平台前缀的字段。
  > `SettingsStore`：`RwLock<Settings>` 内存态 + `app_local_data_dir/settings.json`（dev 文件名带 `.dev` 后缀，与 `WindowStateStore` 同套约定）。`load_from_disk` 主文件解析失败回退 `.bak`，再失败回退 `Default`——单次坏盘不丢全部偏好。`write_atomic`：现盘 rename → `.bak`（顶掉上一份备份），新内容写 `.json.tmp` 后 `fs::rename` 升级为主文件；rename 在同 FS 下是原子的，避免半截 JSON。
  > `update(patch)` 接 `serde_json::Value`，要求 object，走 `deep_merge` 后再 `from_value::<Settings>`——这样前端可以发任意子树（如 `{"clipboard":{"history":{"maxCount":500}}}`）而不必构造完整对象，且类型校验仍在反序列化阶段集中处理。数组按 patch 整体替换（而非追加合并），对 `itemActions` 这类「顺序就是配置」的字段更直觉。
  > 命令层 `commands/settings.rs::{get_settings, update_settings}`。`update_settings` 检测到 patch 含 `shortcuts` 键时顺带调 `shortcut::apply` 重注册——把「写入后副作用」拢在命令层，避免 store 反过来依赖 `shortcut` 模块。
  > **重构副作用：**
  >
  > - `shortcut/mod.rs` 弃用自维护的 `ShortcutBindings` 结构（重复定义、与 settings 各执一词），改为接收 `&settings::Shortcuts`；`ShortcutManager` 退化为只持有 `active: Vec<(action, Shortcut)>` 用于反注册，bindings 真相源现统一在 settings store。
  > - 旧命令 `get_shortcuts`/`update_shortcuts` 删除，前端走通用 `get_settings`/`update_settings`。
  > - `lib.rs` setup 顺序：`settings::init` → db → `shortcut::init(&handle, &settings.shortcuts)`。注：autostart 仍走独立命令（OS 级状态是真相源，不是设置文件镜像），settings.general.autoStart 与 OS 状态的双向同步留到 6.2 处理。
  >   单元测试：`deep_merge` 覆盖「对象递归 / 数组整体替换」、`#[serde(default)]` 覆盖「缺字段回落默认」三例，全部通过。`cargo check` / `cargo test --lib settings::` 干净。

### 6.2 前端绑定

- [x] 前端 Valtio store 仅作为设置的本地镜像，变更经命令写回 Rust
- [x] 启动时从 Rust 拉取设置初始化
  > 新增 `src/types/settings.ts`（与 `src-tauri/src/settings/model.rs` 字段一一对齐的 TS 类型，含 `DeepPartial<Settings>` 作为 patch 类型）+ `src/stores/settings.ts`（`proxy<{value: Settings|null, loaded: boolean}>` + `loadSettings` / `updateSettings`）。`updateSettings` 直接把 Rust 返回的快照覆盖镜像，不在前端做二次合并，避免本地状态与盘上漂移。
  > `main.tsx` 启动期 `await loadSettings()` 后再 `root.render(<App />)`，组件层无需处理 null 态；IPC 通信异常时用 `console.error` 兜底（启动期 log 通道未就绪），并仍渲染——降级到 `loaded=false`，后续组件读 `settingsState.loaded` 决定是否阻塞。
  > 装了 `valtio@2.3.2`。未引入额外 hook 封装，组件层直接 `useSnapshot(settingsState)`——React 19 下 valtio 工作良好，不需要 `forwardRef` 等老式样板。

---

# 阶段 7 · 前端 UI（MVP）

### 7.1 应用骨架

- [x] `App.tsx`：HeroUI Provider + 主题 + i18n + 路由
  > HeroUI v3 不需要 Provider 包裹（参见 https://heroui.com/react/llms-patterns.txt ：「No provider required」），改为在 `index.html` 上声明 `class="light"` 与 `body class="bg-background text-foreground"` 让 HeroUI 的 token/CSS 变量生效，避免后续暗色切换时只改 class 而背景色不跟随。
  > `App.tsx` 仅保留 `RouterProvider`，并加注释把后续 7.1.3 主题与 7.5 i18n 的插入点标出来，避免下次又被「为什么没 Provider」绊一下。
- [x] 全局事件监听 hook `useTauriListen`（封装 Tauri event）
  > `src/hooks/useTauriListen.ts`：走官方 `@tauri-apps/api/event` 的 `listen`（`Promise<UnlistenFn>`）。原方案用 `tauri-awesome-rpc` 的同步 `listen` 免「Promise resolve 时组件已卸载」的竞态，但 awesome-rpc 因依赖链中 `miow 0.2.2` 在 Windows panic 已移除（见 §选型表），改用 `cancelled` 标记拦截迟到的注册。
  > payload 由 `@tauri-apps/api/event` 的 `Event<T>` 携带，hook 用泛型 `T` 让调用方断言；handler 走 ref 转发，effect 仅依赖事件名，业务侧不必 `useCallback`。
- [x] 暗色模式应用（监听系统 / 跟随设置），注入 CSS 变量
  > `src/hooks/useApplyTheme.ts` + `App.tsx` 调用：**两条路径都走** —— ① Tauri `getCurrentWindow().setTheme(null|light|dark)` 让原生 chrome（标题栏装饰、滚动条、原生菜单）跟随；② `html.classList` 写 `light/dark`（HeroUI v3 `@custom-variant dark (&:is(.dark *))` 依赖该 class）+ `data-theme` + `style.colorScheme`。旧版 EcoPaste 就是这么干的——只改 DOM 不改窗口会让原生区域与内容主题割裂。
  > `auto` 模式：`setTheme(null)` 把跟随系统的责任交给 Tauri，订阅 `appWindow.onThemeChanged` 而非 `matchMedia`——单一信源避免双订阅打架；切到显式 light/dark 时立刻 unlisten。effect 依赖 `[theme, loaded]`，未加载完成时不动 DOM/窗口，避免和 `index.html` 默认 `class="light"` 闪一下。
  > 顺手新增 `src/utils/log.ts`：封装 `@tauri-apps/plugin-log` 的 `error/warn/info/debug`，与 Rust `log::error!` 同源（一并进 LogDir 文件 / Stdout / Webview console），catch 兜底落控制台避免 unhandled rejection。`useApplyTheme.ts` + `main.tsx` 全部走 `log.error`，不再散落 `console.error`。capability 加 `log:default` 让前端能调到 plugin-log 的 IPC；npm 装 `@tauri-apps/plugin-log`。

### 7.1.5 剪贴板条目「来源应用」（趁数据库未发版）

- [x] migration 0001：新增 `clipboard_apps`（id/name/icon_file/platform/created_at/updated_at），`clipboard_items` 加 `source_app_id` 外键 + 索引
- [x] `db/apps.rs`：`upsert_app`（UPDATE 失败再 INSERT，保留 created_at）/ `find_app_by_id`
- [x] `clipboard/app_store.rs`：app icon PNG 落 `<app_local_data>/resources/app-icons/<hash[..2]>/<sha256>.png`（sha256 内容寻址，幂等）
- [x] `clipboard/source.rs`：macOS 走 `NSWorkspace.frontmostApplication` 取 bundle id / localizedName / icon（NSImage → TIFF → NSBitmapImageRep → PNG）；Windows 占位返回 None
- [x] watcher：回调里**先**同步抓前台应用（晚一步前台会切回 EcoPaste 自己），materialize 后随 item 一并 upsert；apps 表写失败降级为 source_app_id=None，不阻断条目入库
- [x] `read_clipboard` 命令同走采源链路；新增 `get_clipboard_app_icon_path` 命令（沿用图片同款防穿越校验）
- [x] 待 7.2 接入前端时一并补：capability `asset:default` scope 放开 `app-icons/**`、`ClipboardItem` 的 TS 类型同步 `sourceAppId`
  > Tauri v2 的 asset 协议没有 `asset:default` 这种 capability permission（实测会被 build script 拒绝），真正的开关是：① `Cargo.toml` 加 `protocol-asset` feature；② `tauri.conf.json > app.security.assetProtocol` 配 `enable: true` + `scope: ["$APPLOCALDATA/resources/app-icons/**"]`。scope 就是访问网关，capabilities 不需要再列。
  > TS 类型新增 `src/types/clipboard.ts`，camelCase 与 `db/models.rs` 对齐：`ClipboardItem`（含 `sourceAppId`）/ `ClipboardApp` / `ClipboardItemQuery` / `ReadClipboardResult` / 三个枚举 union。日期字段按 serde 默认 ISO8601 字符串。

### 7.2 剪贴板历史列表（main 窗口）

- [x] 列表容器：`react-virtuoso` 虚拟滚动
  > `pnpm add react-virtuoso`（4.18.7）。`src/pages/Clipboard/components/ClipboardList.tsx` 用 `<Virtuoso>` 渲染列表，`style={{height: "100%"}}` + 外层 `h-screen w-screen` 让其占满 main 窗口（360×600 固定）。Row 仅占位（kind + content slice）—— 分类型卡片是 item 2 的职责。
  > 顺带把数据源 `list_clipboard_items` 也补上（item 3 预热）：`src-tauri/src/commands/clipboard.rs` 薄封装 `db::items::query_items`，参数 `Option<ClipboardItemQuery>`，缺省走 Rust 默认（limit=50, offset=0, createdAtDesc）；FTS 委派沿用 `query_items` 内部判断。`lib.rs` 注册了。初始加载用 `invoke<ClipboardItem[]>("list_clipboard_items")`，失败走 `@/utils/log`（不要裸 console）。事件驱动增量刷新留给 item 3 余下部分。
- [x] 历史项组件：按 type 渲染（text/rtf/html/image/files 各自卡片）
  > 新增 `src/pages/Clipboard/components/cards/`：`ClipboardCard`（按 `kind` 分发 + 共享外壳：相对时间 + 置顶/收藏角标）；`TextCard` 内部再按 `subKind` 分支——`color` 给色块预览、`url`/`email` 着 primary 色、`rtf`/`html` 读 `searchText`（Rust 入库已落纯文本，富文本/HTML/Markdown 的真正渲染留给 item 5）；`ImageCard` 仅展示尺寸 + 体积（`<img>` 预览要 asset scope `images/**`，留给 item 5）；`FilesCard` 按换行解析 `content`（与 Rust `write_files` 一致），展示 basename + 总数。
  > `ClipboardList.tsx` Row 改为 `<ClipboardCard item={item} />`。没引入 HeroUI 组件——本步全用 Tailwind + HeroUI 的 token 色（`text-default-500`、`border-divider`、`text-primary`），先把分发骨架稳住，Card/Chip 等组件留到后续操作按钮（item 4）一起接。
- [x] 从 Rust 命令拉取分页数据；监听「剪贴板更新」事件增量刷新
  > 抽出 `src/pages/Clipboard/hooks/useClipboardItems.ts`：分页 50 条/页，Virtuoso `endReached` 触发 `loadMore`（offset 取当前 `items.length`，`loadingRef` 抑制重入，返回不足 PAGE_SIZE 时把 `hasMore` 落为 false）；订阅 `clipboard://updated`（payload `{id, deduplicated}`）走新的 `get_clipboard_item` 命令拉单条，setItems 时先按 id 过滤再前置 —— 这样去重场景（旧条目 updatedAt 更新）会自然「移到顶部」，新条目同路径处理为「直接前置」，避免整页 refetch 打断滚动。
  > Rust 侧加 `get_clipboard_item(id) -> Option<ClipboardItem>` 薄封装 `find_item_by_id`，`lib.rs` 注册。`ClipboardList.tsx` 瘦身为纯 Virtuoso，状态全在 hook。事件名常量直接写在 hook 里（不引 Rust 常量到前端，避免跨端硬耦合）。
- [x] 操作：复制回 / 粘贴 / 收藏 / 删除 / 备注（调用 Rust 命令）
  > Rust 侧新增 3 个薄封装命令：`toggle_clipboard_item_favorite` / `delete_clipboard_item` / `update_clipboard_item_note`（lib.rs 注册）。`update_clipboard_item_note` 把空串 / 全空白归一化为 NULL，保证「无备注」在库里只有一种表示。复制/粘贴沿用已有的 `write_to_clipboard` / `paste_clipboard_item`。
  > 前端 `useClipboardItems` 暴露 `actions: ClipboardActions`（copy/paste/toggleFavorite/remove/updateNote），写操作走乐观更新（先动本地、失败仅 log，不回滚——剪贴板/SQLite 出错本来罕见，整页 refetch 反而打断滚动；后续接 toast 时统一在 hook 里挂）。`ClipboardCard` 加 hover 显形的文字按钮行（粘贴/复制/收藏/备注/删除）+ 折叠 textarea 备注编辑器，按钮统一 `e.stopPropagation()` 避免冒泡到后续 item 6 的行选中。`ClipboardList` 把 actions 通过 prop 透传给卡片。
- [x] HTML 预览（DOMPurify sanitize）、Markdown 渲染、RTF 渲染、图片预览
  > `tauri.conf.json > assetProtocol.scope` 追加 `$APPLOCALDATA/resources/images/**`，让缩略图能走 `asset://`。`ImageCard` 在 useEffect 里 invoke `get_clipboard_image_path`（thumbnail=true）拿到磁盘路径后 `convertFileSrc` 拼成 asset URL，`<img>` 渲染 + 占位框；失败仅 log 不抛。`TextCard` 对 `subKind=html` 走新增 `HtmlPreview` 子组件：`DOMPurify.sanitize`（FORBID_TAGS: script/style）后 `dangerouslySetInnerHTML`，外层 `max-h-16 overflow-hidden` 防大段 HTML 撑爆行。RTF 沿用 `searchText` 纯文本预览（不引 RTF→HTML 库，依赖太重且现有库质量参差）；Markdown 暂跳过（Rust detect.rs 没识别 subKind=markdown，需要时再补检测+marked）。
- [x] 选中态、键盘上下选择、Enter 粘贴、Esc 隐藏
  > 主窗口 `tauri.conf.json` 加 `focusable: false`：macOS 自动用 NSPanel；Windows 不抢前台焦点但 WebView 收不到键盘事件——参考 QuickClipboard 的方案在 Rust 侧加 `keyboard/` 模块装 `WH_KEYBOARD_LL` 低级钩子（用现有 `winapi` crate，无新依赖），命中 Up/Down/Enter/Esc 后 emit `keyboard://nav` `{action}` 给前端，并返回 `LRESULT(1)` 吞键 + `CONSUMED_KEYS` 集合让配对的 KEYUP 也吞，防止泄到背后聚焦的应用。钩子线程跟随主窗口 show/hide 生命周期：`show_window`/`hide_window` 在 main 分支调用 `enable/disable_navigation_keys`，disable 通过 `PostThreadMessageW(WM_QUIT)` 让消息泵自然退出，避免常驻全局抓键。macOS 走 noop 实现。`show_window` 同时跳过对 main 窗口的 `set_focus`（focusable=false 后无意义）。前端新增 `useListNavigation` hook 同时挂 `window.keydown`（macOS 路径）和 `useTauriListen(keyboard://nav)`（Windows 路径），另一端自然不触发，不做平台判断；`ClipboardList` 接入 `selectedIndex` + `VirtuosoHandle.scrollToIndex` 跟随，Enter → `actions.paste`，Esc → `invoke("hide_window", {label: "main"})` 复用已有命令。`ClipboardCard` 新增 `isSelected` prop 加 `bg-default-100` 视觉态。

### 7.3 搜索

- [x] 搜索框组件（位置 top/bottom 可配）
  > 新增 `src/pages/Clipboard/components/SearchBar.tsx`：HeroUI v3 `SearchField` 复合组件（Group + SearchIcon + Input + ClearButton），受控 `value` / `onChange`，aria-label 兜底。`Main/index.tsx` 用 `useSnapshot(settingsState)` 读 `clipboard.search.position`（缺省 `top`），外层 `flex flex-col` + `flex-col-reverse` 切换 top/bottom；list 容器套 `min-h-0 flex-1` 把剩余高度让给 Virtuoso，避免 search bar 把列表挤出可视区。`keyword` 暂存在 `Main` 本地 state，下面 item 2 接 Rust `list_clipboard_items({ keyword })` 时再下沉到 `useClipboardItems`。
- [x] 输入 → 调 Rust 列表查询命令（`ClipboardItemQuery.keyword` 带关键词时自动走 FTS5）→ 渲染
  > `useClipboardItems(keyword)` 接受外部关键词：`Main` 持有 `keyword` state，`SearchBar` 与 `ClipboardList` 共用。useEffect 用 `setTimeout(200ms)` 防抖 + `cancelled` 标志拦下旧响应（防抖中 keyword 再变 → clearTimeout 撤掉；已发请求期间 keyword 再变 → cancelled 阻止旧结果覆盖新结果）。`buildQuery` 把 trim 后非空的 keyword 放进 `ClipboardItemQuery`（Rust 端 `query_items` 看到 `keyword` 即委派 `search_items_fts` 走 FTS5）。`activeKeywordRef` 给 `loadMore` 和 `clipboard://updated` 回调读当前生效关键词，搜索态下跳过 live 事件——搜索结果是关键词快照，新条目未必匹配，硬塞会污染结果；清空搜索时 effect 重新拉取，最新条目自然回顶。
- [x] 命中文本高亮（react-mark.js 等价）
  > 新增 `src/pages/Clipboard/components/Highlight.tsx`：纯组件实现，避免引第三方库——`escapeRegExp` 转义后大小写不敏感切分，奇数索引片段 `<mark>` 包裹（HeroUI `bg-warning-soft` token）。key 用 `${i}-${part}` 既稳又避撞。keyword 从 `Main` → `ClipboardList` → `ClipboardCard` → `TextCard` 透传，TextCard 在非 HTML 分支（含 url/email/color/path/rtf 的纯文本预览）包裹 `<Highlight>`。HTML 预览跳过——sanitize 后是已渲染 DOM，节点级别注入 `<mark>` 复杂度过高且收益小，需要时再单独处理。FilesCard 用 basename 不参与 FTS 匹配，也跳过。
- [x] 默认聚焦 / 自动清空（按设置）
  > Rust `window/mod.rs` 在统一入口处加 `window://visibility` 事件（payload `{label, visible}`），`show_window` / `hide_window` 成功后 emit——不复用 `tauri://focus`/`blur`，因为 macos NSPanel 走自定义 emit、Windows 主窗口为不抢焦点设计，平台行为不一致。`SearchBar` 加 `inputRef` 转给 `SearchField.Input`。`Main` 监听 `window://visibility`：`label==="main"` 时按 `settings.clipboard.search.defaultFocus` 决定 `focus()`，按 `clearOnHide` 决定 `setKeyword("")`。两项均走 `useSnapshot` 读最新设置，关掉再开即时生效。

### 7.4 偏好设置（preference 窗口）

- [x] 设置页框架（分组：常规 / 剪贴板 / 快捷键 / 外观 / 关于）
  > `src/pages/Preference/index.tsx` 用 HeroUI v3 `Tabs orientation="vertical"`：左侧 `w-36` 侧栏 + 右侧 `flex-1 overflow-auto` 内容区，撑满 `h-screen w-screen`（preference 窗口 700×480）。`GROUPS` 数组单一来源驱动 Tab 与 Panel，新增/调整分组只改一处。各 Panel 暂为「待接入」占位 —— 控件绑定 `get_settings`/`update_settings` 留给下一项；文案先硬编码中文，等 7.5 引 i18n 后统一替换。
- [x] 各项控件绑定 Rust `get_settings`/`update_settings`
  > 新增 `components/Row.tsx`（label + 描述 + 右侧 control 行）和 `components/Field.tsx`（`Toggle` 薄封装 HeroUI Switch；`SelectControl<T>` 薄封装 Select+ListBox，泛型锁定枚举字面量），避免每个面板重复写 `Switch.Control/Thumb` / `Select.Trigger/Popover/ListBox.Item`。`panels/GeneralPanel.tsx` 4 个 Switch；`panels/ClipboardPanel.tsx` 按 content/history/search/window/feedback 5 个 Section 拆分（每段标题 + Separator 分隔），含 `NumberField` 处理 `history.maxCount` 与 `retention.value`（unit=forever 时隐藏数值行）；`panels/AboutPanel.tsx` 用 `@tauri-apps/api/app` 的 `getName/getVersion` 异步取值 + 仓库链接。`shortcuts` / `appearance` 暂留占位指向 7.4 第 3、4 项。所有写操作走 `updateSettings({ <group>: { <field>: v } })`，依赖 Rust `update_settings` 的 JSON 合并把单字段补丁深合并进 `Settings`，再用返回快照覆盖 `settingsState`——前端不本地变更、不回滚，避免与 Rust 真相源漂移。`itemActions` 顺序编辑器复杂度高（拖拽 + 子集选择）且非 MVP 关键路径，留作后续单独项。各面板用 `useSnapshot(settingsState).value`，未加载时 `return null`（preference 独立窗口启动时 `main.tsx` 已发起 `loadSettings`，首帧短暂空白可接受）。
- [x] 快捷键录制控件
  > 新增 `components/ShortcutInput.tsx`：触发按钮显示当前 binding（空值显占位「未设置」）；点击展开 HeroUI `Popover`，内含 dashed 边框的「录制区」按钮 + 清除/取消/保存。`useEffect(open)` 在 popover 打开后 `focus()` 录制按钮（不用 autoFocus，绕开 biome a11y）。`onKeyDown` 里 `preventDefault + stopPropagation` 吞下所有按键，按 `event.code` 映射成 `Letter/Digit/Fn/Arrow/Space|Enter|…`，再按平台拼修饰键名（mac: Cmd/Option；其他: Super/Alt + Ctrl/Shift）；提交时 `toCanonical` 把 Option 写回 Alt，库内统一一套字符串，避免分平台两份。完整组合（≥1 修饰 + 1 主键）在 keydown 即提交并关闭 popover；`modifierOnly` 模式（QuickPaste.modifier）改为按 Save 按钮提交——因为 keyup 阶段 metaKey/ctrlKey 已清零，无法在那时取到组合，必须显式确认。`panels/ShortcutsPanel.tsx` 接入 4 行：openClipboard / openPreference / pastePlain（不在 OS 级注册，但格式一致复用同组件）/ quickPaste.modifier（modifierOnly）。写操作走 `updateSettings({ shortcuts: { ... } })`，Rust 端 `commands/settings.rs` 检测到 `shortcuts` 字段会自动 `shortcut::apply()` 重新注册，冲突走 `shortcut://conflict` 事件——前端 toast 系统暂缺，这里不订阅，等通知组件就位再补；命令行/Rust 日志已能看到冲突原因。浏览器层面有少量系统级组合（如 macOS Cmd+Space）拿不到 KeyboardEvent，属已知限制。
- [x] 主题切换、语言切换
  > `panels/AppearancePanel.tsx` 两个 `SelectControl`：主题 auto/light/dark、语言 zh-CN/en-US。主题写回 `updateSettings({ appearance: { theme } })`，App.tsx 已挂的 `useApplyTheme` 监听 `settingsState.appearance.theme`，自动重新 `appWindow.setTheme + applyClass`——预览/主窗都即时切。语言这一项只落库，实际文案切换等 7.5 引 i18next；当前 description 注明「i18n 接入后即时生效」。

### 7.5 i18n

- [x] 引入 i18next + react-i18next
  > `pnpm add i18next react-i18next` 引入。`src/locales/index.ts` 暴露 `initI18n(language)` —— 用 `initReactI18next` 注册资源、`fallbackLng: "zh-CN"`、`interpolation.escapeValue: false`（React 已自带 XSS 防护）、`returnNull: false`（缺翻译返回 key 字符串便于排查）。`src/main.tsx` 在 `loadSettings()` 之后、首屏 `render` 之前调用 `initI18n(settings.appearance.language)`，IPC 失败时降级到 zh-CN。新增 `src/hooks/useApplyLanguage.ts`，类似 `useApplyTheme` 订阅 `settingsState.appearance.language`，运行时切换走 `i18n.changeLanguage`；`App.tsx` 同时挂 `useApplyTheme` 和 `useApplyLanguage`。Tauri 系统 locale 作初始值（第 3 项）暂未做，目前以 Rust 默认 `Language::ZhCN` 起步。
- [x] 文案表：zh-CN（默认）/ en-US
  > `src/locales/{zh-CN,en-US}.json` 两份，单 `translation` namespace + 层级 key（如 `clipboard.autoPaste.option.disabled`）。`Language` 类型 / Rust `Language` 枚举只保留 `zh-CN` + `en-US` 两个字面量，磁盘上若残留旧值（zh-TW/ja-JP）由 store 的 `.bak/Default` 兜底回到默认 zh-CN。所有 Preference 面板（General/Clipboard/Shortcuts/Appearance/Preference index）与 `ShortcutInput` 已全部替换 `useTranslation` + `t()`，硬编码中文清零。
- [x] 语言可由 Rust 提供系统 locale 作初始值（`tauri-plugin-os`/locale）
  > 用轻量 `sys-locale` crate（tauri-plugin-os 内部用的同一份）替代整套 plugin，避免给前端再开一条 OS 信息读取通道。`Language::from_system_locale(tag)` 做粗匹配：任何 `zh-*` 归 zh-CN，其余 en-US。`SettingsStore::new` 区分「真首次启动（主文件 + `.bak` 均不存在）」和「文件存在但坏掉」两种路径 —— 只有前者会读取系统 locale 并立刻 `write_atomic` 落盘，确保第二次启动起就走常规分支、用户手动改过的语言也不会被系统 locale 覆盖。坏文件路径仍回退到 `Settings::default()`（zh-CN），避免一次坏盘静默改语言。`load_from_disk` 返回 `Option<Settings>` 表达「首次」与「找不到可读副本」的差别。

> **MVP 里程碑**：到此可完成「监听→入库→列表展示→搜索→粘贴→设置」闭环。

---

# 阶段 8 · 增强功能（MVP 后）

### 8.1 历史清理后台任务（Rust）★

- [x] 后台定时任务：按 `duration` + `maxCount` 清理（保留置顶 / 收藏）
  > `db/items.rs::cleanup_history(pool, older_than, max_count)` 两条 DELETE：① `is_pinned = 0 AND is_favorite = 0 AND created_at < cutoff` 按时间砍；② `id IN (SELECT id ... WHERE is_pinned = 0 AND is_favorite = 0 ORDER BY created_at DESC LIMIT -1 OFFSET max)` 按条数砍——置顶 / 收藏不计入名额也不会被删。`clipboard/cleanup.rs::spawn` 启动即跑一次，然后 `tokio::time::interval(60min)` 周期触发；每个 tick 从 `SettingsStore::snapshot()` 取最新 `history`，`retention_cutoff` 把 `Retention { value, unit }` 折算成 `DateTime<Utc>`（`Forever` / `value == 0` 视为禁用，月按 30 天近似）。清掉非零行后 `emit(CLIPBOARD_UPDATED_EVENT, {cleanup: n})` 让前端列表刷新。挂在 `clipboard::watcher::init` 里，复用同一份 `SqlitePool::clone`。
- [x] 启动时执行一次 + 周期执行
  > 见上：`clipboard/cleanup.rs::spawn` 在 `tauri::async_runtime::spawn` 内先调用 `run_once`，再用 `tokio::time::interval(60min)` 进入周期循环；interval 首个 tick 立即返回故丢弃一次，避免与启动那次重复跑。`SettingsStore` 未 manage 时 `run_once` 直接 return（保守降级）；清理失败仅 `log::warn` 不中断循环，确保单次抽风不影响后续 tick。

### 8.2 分组 / 收藏视图

- [x] 前端分组 tab、收藏过滤；Rust 查询支持 group/favorite 过滤
  > Rust：`db/items.rs::query_items` 早已支持 `favorite`/`group_id` 过滤；本步只补 `list_clipboard_groups` 命令（`commands/clipboard.rs`），薄封装 `db::groups::list_groups`，在 `lib.rs::invoke_handler` 注册。前端：① 新 store `stores/clipboardView.ts` 用 Valtio 持有 `tab: { kind: "all" | "favorite" | "group", groupId? }`，带 `tabToKey`/`keyToTab` 双向编码（`"group:<id>"`）。② 新 hook `useClipboardGroups` 一次性 `invoke("list_clipboard_groups")`——暂无分组 CRUD UI 故不订阅事件。③ `useClipboardItems` 增加 `tab` 参数：`buildQuery` 把 tab 折算成 `favorite`/`groupId` 过滤；effect 依赖 `[keyword, tab]`，切 tab 立即 refetch；`tabRef` 让 `loadMore` 和事件回调读最新视图。④ `clipboard://updated` 事件：在「收藏」/「分组」视图下，对拉到的单条用 `matchesTab(item, tab)` 判定，不匹配则不前置（避免普通新条目污染收藏视图）。⑤ `toggleFavorite` 乐观更新：在收藏视图下取消收藏直接从本地列表移除，UI 立刻反映过滤集合变化。⑥ 新组件 `ClipboardTabs` 用 HeroUI `Tabs`（横向）渲染「全部 / 收藏 + 各分组」，挂在 `SearchBar` 下方；i18n 加 `clipboard.tabs.{all,favorite,ariaLabel}` 两份。分组 CRUD UI（新建 / 重命名 / 删除）留到 8.3 自动行为或后续单做。

### 8.3 自动行为

- [x] auto-favorite（按规则自动收藏，Rust 判定）
  > 沿用旧版「写入备注后自动收藏」语义，但判定下沉到 Rust，避免前端来回读设置。`db/items.rs::mark_item_favorite` 新增幂等 setter（区别于 `toggle_item_favorite` 的翻转），已收藏的不变；命令层 `update_clipboard_item_note` 在归一化后的 note 非空时，从 `SettingsStore.snapshot().clipboard.content.auto_favorite` 读最新开关，开启则调用 `mark_item_favorite`。清空备注（normalized = None）不触发，比旧版「保存即收藏」更直觉。命令返回值从 `()` 改为 `bool`（是否触发了 auto-favorite），前端 `useClipboardItems.updateNote` 据此把乐观更新里的 `isFavorite` 一并设为 true，否则列表会与库不一致（在「收藏」视图下尤其明显）。`SettingsStore` 通过 `app.try_state` 取，未 manage 时保守视为关闭。
- [x] auto-paste 模式（never / 双击 / 直接）
  > 设置 / 命令 / i18n 都已就绪（`Settings.clipboard.content.auto_paste` 三态 `disabled` / `singleClick` / `doubleClick`，命令 `paste_clipboard_item` 写回+模拟 ⌘V/Shift+Insert），缺的是前端把 setting 与列表行点击挂上钩。在 `ClipboardList.tsx` 用 `useSnapshot(settingsState)` 拿到 `autoPaste`，把每个 Virtuoso item 外套一层 `<div>` 挂 `onClick`/`onDoubleClick`：单击一律先 `setSelectedIndex(idx)` 保持键盘导航游标与点击位置一致，再按模式决定是否 `actions.paste(item.id)`；双击仅在 `doubleClick` 模式触发。`disabled` 模式只选中不粘贴，与设置面板的「仅选中」文案对齐。卡片内 `ActionButton` 早已 `stopPropagation`，所以 hover 工具条（粘贴/复制/收藏/备注/删除）不会被行级 onClick 串扰；`noteOpen` 的 textarea 区同样 stopPropagation。`autoPaste` 取不到（设置未加载）时默认按 `doubleClick`，与 Rust 端 `Settings::default()` 一致避免首屏闪烁两种行为。
- [x] auto-sort（时间 / 频率）
  > 底层早就齐了：`db/items.rs::query_items` 的 `ClipboardItemSort::{CreatedAtDesc, UseCountDesc}` 已实现并带测试，`upsert_item` 命中去重时把 `use_count += 1`（不在 paste/copy 时累加，「频率」== 复制重复次数）；设置位 `Settings.clipboard.content.auto_sort_by_frequency: bool` 与 ClipboardPanel 的开关、i18n 文案都在。缺的是前端拉列表时把这个 setting 折算成 `sort` 参数。改两处：① `useClipboardItems` 增加 `sort: ClipboardItemSort` 参数（默认 `createdAtDesc` 与 Rust `Default` 对齐），`buildQuery` 注入 `sort`，effect 依赖加入 `sort` 切换时整页 refetch，`loadMore` 走 `sortRef` 避免闭包陷旧值。② `ClipboardList` 用 `useSnapshot(settingsState)` 读 `autoSortByFrequency`，映射成 `useCountDesc` / `createdAtDesc` 传入 hook。`clipboard://updated` 事件仍按「前置插入」处理：在频率排序下严格按 use_count 重排会破坏滚动位置，dedup 的条目刚 +1 大概率本就靠前，下次显式刷新（切 tab / 关键词 / 切换排序）会重新排序，符合预期。

### 8.4 声音通知

- [x] 复制成功提示音（前端播放或 Rust 触发）
  > 下沉到 Rust 监听链路：`clipboard/sound.rs::maybe_play_copy` 在 `persist_and_notify` upsert 之后调用，从 `SettingsStore.snapshot().clipboard.feedback.copy_sound` 读最新开关，关闭则直接 return。播放本身用 `rodio`（`default-features = false, features = ["symphonia-mp3"]`）解码 `assets/sounds/copy.mp3`（旧版同款，`include_bytes!` 烤进二进制，24KB 不值得走 resource 路径 + 文件 IO）。`OutputStream` 是 `!Send` 且必须存活到播放结束，所以每次播放都新开短命 `std::thread`：建流 → 解码 → `sink.sleep_until_end()` → drop；剪贴板事件频率远低于建流开销（ms 级），不必维护常驻 worker。播放失败仅 `log::warn`，不阻断入库主流程。自身写回触发的事件在 `guard.should_skip` 处已被抑制，不会响——只有真正的用户复制才会播。`SettingsStore` 未 manage 时保守视为关闭，与 `cleanup.rs` 一致。

### 8.5 多选删除

- [ ] 列表多选模式 + 批量删除（Rust 命令 + 前端 UI）
  > Rust：`db/items.rs::delete_items_bulk(pool, ids: &[i64])` 用 `DELETE ... WHERE id IN (?, ?, ...)` 一条 SQL 删掉，事务包裹保证原子性；附带删除其图片 / 文件副本（复用 `delete_item` 的清理逻辑，提到 `cleanup_item_assets` 私有 helper 复用）。命令 `delete_clipboard_items(ids: Vec<i64>)` 薄封装，删完 `emit(CLIPBOARD_UPDATED_EVENT, {bulk_delete: n})` 让列表刷新。前端：① `stores/clipboardView.ts` 增 `selectionMode: boolean` + `selectedIds: Set<number>`，提供 `toggleSelection` / `clearSelection` / `enterSelectionMode` actions。② `ClipboardList` 进入多选模式后行点击改为 toggle 选中（不再触发 paste），渲染左侧 checkbox；键盘导航暂保留单选高亮。③ 工具栏（`ClipboardTabs` 旁或 `SearchBar` 内）显示「全选 / 反选 / 删除选中(N) / 退出」。④ 删除前用 HeroUI `Modal` 二次确认，删除后清空选区并退出多选。⑤ 快捷键：列表聚焦时 `Cmd/Ctrl+A` 全选，`Esc` 退出多选。

### 8.6 内容加密保护

- [ ] 给单条剪贴板内容设置访问密码（与备注独立）
  > 需求：用户对敏感条目设密码，列表中默认遮蔽内容，输入正确密码后短时解锁查看 / 复制 / 粘贴；密码与「备注」是两套独立字段。
  > 数据层：`clipboard_items` 加两列 `password_hash TEXT`（argon2 哈希，nullable）+ `password_hint TEXT`（可选提示，nullable）；migration 写入 `0001_init.sql`（仓库未发版，按现有约定直接改 init）。`models.rs::ClipboardItem` 同步加 `is_locked: bool`（由 `password_hash IS NOT NULL` 推导，**前端不下发 hash**），返回给前端的 payload 只带 `is_locked` + `password_hint`。
  > Rust 命令（`commands/clipboard.rs`）：① `set_item_password(id, password, hint)` 用 `argon2` crate 算 hash 存库；密码空字符串视为「清除」（hash / hint 置空）。② `verify_item_password(id, password) -> bool` 校验，成功返回 true，失败返回 false（不抛错）。③ `read_locked_item(id, password) -> ClipboardPayload` 校验密码后返回完整 payload（包含被加密的 text/html/files 等明文字段）；失败返回 `AppError::InvalidPassword`。④ 锁定条目走 `paste_clipboard_item` / `copy_clipboard_item` 时强制要求先 `verify`，否则拒绝。
  > 内容存储策略：MVP 版**不加密磁盘明文**，仅加「密码门禁」——`is_locked = true` 的条目，列表 payload 里 `text` / `html` / `files` 等敏感字段返回 `None`（DB 仍是明文，靠命令层过滤），前端拿不到内容；调用 `read_locked_item` 才下发明文。后续若要真加密落盘，再加 `encrypted_blob` 列 + AES-GCM（key 由用户密码 KDF 派生），属于增强项不在本步。
  > 前端 UI：① 卡片右上 `ActionButton` 增「设置密码」（图标：lock），调出 `Modal` 输入密码 + 提示，确认后 `invoke("set_item_password", ...)`。② 锁定条目卡片：内容区替换为占位（图标 + hint），点击触发 `Modal` 输密码 → `verify_item_password`；通过后把明文缓存到 `clipboardViewStore.unlockedIds: Set<number>`（仅当前会话生效，关窗清空），渲染时若在集合内则正常展示。③ 复制 / 粘贴动作在锁定条目上点击同样先弹密码框；通过后调用 `read_locked_item` 拿明文 → 写回剪贴板。④ 备注与密码并存：解锁后才能看 / 改备注（备注本身不加密但跟内容同入口）。⑤ i18n 加 `clipboard.password.{set,unlock,verify,hint,placeholder,wrong,clear}` 两份。
  > 安全注意：argon2 参数走默认即可（m=19MiB, t=2, p=1）；`verify` 命令做简单速率限制（同一 id 连续 5 次失败后 30s 内拒绝，记到内存 `DashMap<i64, FailState>`，不持久化），防本机暴破。

### 8.7 贴边隐藏（Edge Hide）

- [ ] 主窗口拖到屏幕边缘时自动收起，鼠标贴边唤出
  > 旧版（macOS NSPanel + 自研 eco-window）有此特性，重构版把窗口位姿与贴边判定都下沉到 Rust。
  > Rust：`window/edge_hide.rs` 维护 `EdgeHideState { docked: Option<Edge>, hidden: bool, peek_strip_px: i32 }`，`Edge = Top|Left|Right`（不做 Bottom，避免与 macOS Dock / Windows 任务栏冲突）。① 监听窗口 move 结束事件（macOS `NSWindowDidMoveNotification` / Windows `WM_EXITSIZEMOVE`）：如果窗口某条边距离当前屏 visibleFrame 对应边 ≤ 8px，标记 `docked = Some(edge)`，触发 `slide_out`——把窗口移到只剩 `peek_strip_px = 4` 像素挂在屏幕外，`hidden = true`。② 在 docked 状态下启动一个 OS 级鼠标位置轮询（macOS `NSEvent.mouseLocation`，Windows `GetCursorPos`，间隔 50ms，使用 `tokio::time::interval`，仅在 docked 时跑，未 docked 时停掉 task）：鼠标进入「贴边触发带」（沿对应边、宽 4px、长度 = 屏幕该边长度）时 `slide_in` 恢复原位置；窗口失焦且鼠标离开窗口区域 ≥ 300ms 后再次 `slide_out`。③ 拖动远离边缘（距离 > 32px）解除 docked。④ 滑入 / 滑出做 150ms 缓动（`tokio::time::sleep` + 分帧 `set_position`，10 帧足够顺滑；可选先做硬切版，缓动留作打磨）。
  > 设置：`Settings.window.edge_hide_enabled: bool`（默认关，避免对新用户造成困惑），`window/edge_hide.rs` 启动时按设置 enable / disable 整个模块；`SettingsStore::subscribe` 收到变更时停 / 启监听 task。设置面板 `Preference/panels` 加开关 + 简短说明文案。
  > 命令：`set_edge_hide_enabled(enabled)` 仅薄封装 `SettingsStore::update`；窗口当前是否 docked 不暴露给前端（纯 Rust 内部状态）。
  > 平台注意：macOS 主窗目前是 NSPanel（`focusable=false`），slide_in 时不要抢焦点（保持 panel 行为）；Windows 上注意 DPI 缩放——`peek_strip_px` 用 `monitor.scale_factor()` 换算物理像素再 `set_position`，否则高 DPI 屏会肉眼看不到挂出的 strip。多屏切换：用窗口当前所在 `Monitor::work_area()` 而非主屏，dock 后用户拖窗到另一屏的逻辑在「拖动 → 解除 docked」一步已覆盖。
  > 与现有逻辑联动：用户按全局快捷键 / 托盘呼出窗口时，若 `hidden = true` 直接走 `slide_in`（视为唤出）；窗口隐藏（`hide_main_window`）时取消 docked 计时器，下次 show 重新依设置启用。

### 8.8 系统级 Popover 预览

- [ ] 选中条目按空格 或 鼠标悬停触发独立预览窗口（macOS Quick Look 风格）
  > 需求对标 macOS Finder 空格预览：列表中焦点条目按 `Space` 或鼠标悬停 ≥ 600ms 时弹一个独立的、跟随主窗的浮层窗口展示完整内容，再次按 `Space` / 鼠标移开 / `Esc` 关闭。是「系统级窗口」而非主窗内 Popover——内容长 / 图片大时主窗内放不下，独立窗口可以拉到屏幕剩余空间。
  > Rust：新窗口 label = `clipboard-preview`，用 `WebviewWindowBuilder` 在 `core/setup` 里预创建（hidden、`skip_taskbar`、`always_on_top`、macOS 上 `decorations(false)` + 自绘圆角阴影、Windows 上 `transparent(true) + decorations(false)`），URL `/#/preview` 复用同一 bundle 的另一路由。`window/preview.rs::show_preview(item_id, anchor: Rect)` 计算 popover 位置：优先放主窗右侧，空间不够则左侧 / 下方；尺寸自适应内容类型上限（text 480x600，image 按原始比例最长边 ≤ 720，files 列表 480x400）。`hide_preview()` 仅 `window.hide()` 不销毁（避免 webview 重建抖动）。
  > 命令：`open_clipboard_preview(item_id, anchor: { x, y, width, height })` / `close_clipboard_preview()`；`anchor` 由前端在列表项 DOMRect + 主窗 outer position 算出，传屏幕绝对坐标。预览窗加载完成后通过 `invoke("get_clipboard_payload", { id })`（已存在）拉内容，不另开数据通道；锁定条目（见 8.6）按设计**不允许预览**，前端在 `Space` / hover 触发处过滤掉 `is_locked && !unlocked`。
  > 前端：① `stores/clipboardView.ts` 增 `previewId: number | null`、`hoverIntentId`（带计时器）。② `ClipboardList` 在选中态变化或鼠标进入某行时启动 600ms 定时器；定时到 / 按 `Space` → 计算 anchor → invoke `open_clipboard_preview`；鼠标离开 / 滚动 / 切换选中 / 再按 `Space` / `Esc` → `close_clipboard_preview`。③ 多选模式（8.5）下禁用 hover 预览，避免误触；`Space` 仍可用作单条预览。④ 新页面 `pages/Preview/index.tsx` 一份精简渲染：text 用 `<pre>`，html 走 DOMPurify，image 直接 `<img>`，files 列表表格化；不带交互按钮（复制 / 粘贴留在主窗工具条）。⑤ 路由 `router/index.ts` 加 `/preview` 项，仅在该窗口加载，不出现在主窗导航。
  > 视觉：背景色用 HeroUI `bg-content1` + 12px 圆角 + 主题阴影；macOS 下加细边框（dark / light 各一档）模拟系统弹窗。窗口失去焦点（点其他应用）立即关闭——通过 `WindowEvent::Focused(false)` 监听。
  > 性能：hover 触发用前端 `setTimeout`，不要每次 hover 都走 IPC；预览窗仅维持一份 webview，复用切换内容。设置位 `Settings.clipboard.preview.{enabled, hover_delay_ms}`（默认 enabled=true, delay=600），关闭后 `Space` 也不响应。

### 8.9 Windows 接管 Win+V

- [ ] Windows 平台用 `Win+V` 唤出 EcoPaste，替代系统自带剪贴板历史
  > Windows 系统自带 `Win+V` 剪贴板历史，需要让 EcoPaste 抢占这个组合键。`tauri-plugin-global-shortcut` 注册 `Win+V`（即 `Super+V`，rdev/global-hotkey 表述为 `Meta+V`）大概率失败——因为这是系统保留键；旧版 EcoPaste 同样踩过。
  > Rust（`src-tauri/src/shortcut/windows.rs`，`#[cfg(target_os = "windows")]`）：① 先尝试 `RegisterHotKey(NULL, id, MOD_WIN, 'V')`（`windows` crate `Win32::UI::Input::KeyboardAndMouse`），它能抢在系统剪贴板历史之前；成功后开独立线程跑 message loop 等 `WM_HOTKEY`，收到就 `app.emit("shortcut://toggle_main")` 或直接调 `window::toggle_main_window`。② `RegisterHotKey` 失败（极少数情况）走兜底：用现有 `keyboard/windows.rs` 的低级键盘钩子（`SetWindowsHookExW(WH_KEYBOARD_LL)`）拦截 `VK_LWIN/RWIN + V` 组合——按下时返回非零阻断系统默认处理（这才能压住系统的 Win+V 面板），松开正常放行；命中后同样触发 toggle。③ 系统是否启用了「剪贴板历史」（`HKCU\Software\Microsoft\Clipboard\EnableClipboardHistory`）不影响接管，但启动时检测一次，若开启则 `log::info` 提示用户在设置里关闭以避免视觉冲突（不主动改注册表）。
  > 设置：复用现有「全局快捷键」配置项——给「唤出主窗」快捷键的默认值在 Windows 下设为 `Win+V`（macOS 沿用现有默认，如 `Cmd+Shift+V`）；用户在设置面板里可改成其他组合，改成非 `Win+V` 时拆掉低级钩子兜底、仅走 `tauri-plugin-global-shortcut` 常规注册。
  > 与现有模块联动：`keyboard/windows.rs` 的低级钩子已经在跑（用于 focusable=false 主窗时转发方向键），扩展它的事件分发逻辑即可，不要重复装两个 `WH_KEYBOARD_LL` 钩子（系统对单进程多钩子的顺序无保证，容易竞态）。`shortcut/mod.rs` 在注册阶段先检测组合是否等于「Win+V」，是则跳过 plugin 注册、走 `windows.rs::register_win_v`；否则走原 plugin 流程。
  > 注意：不要尝试禁用系统剪贴板历史功能本身（涉及改注册表 / 组策略，越权且易触发杀软告警）；接管只通过抢注热键 + 低级钩子拦截即可，用户保留通过卸载 EcoPaste 立即恢复系统默认的能力。macOS 不实现本项（macOS 无对应保留键）。

### 8.10 首次启动引导

- [ ] 独立引导窗口，分步带用户走完关键设置（欢迎 → 权限 → 快捷键 → 忽略应用 → ...）
  > 首次启动 / 设置缺失关键项时弹出独立窗口走完引导；完成后写入 `Settings.onboarding.completed = true`，后续不再自动弹（仍可从设置面板手动重开）。
  > Rust：新窗口 label = `onboarding`，`core/setup` 启动时按 `SettingsStore.snapshot().onboarding.completed` 决定是否创建：未完成则用 `WebviewWindowBuilder` 创建（800x560，居中，`resizable(false)`，普通装饰，URL `/#/onboarding`），主窗 / 托盘按需仍正常初始化但不抢焦点；已完成直接不创建。命令 `open_onboarding()` 用于设置面板里的「重新查看引导」按钮，存在窗口就 `set_focus`，否则现建。
  > 数据模型：`Settings.onboarding: { completed: bool, last_step: u8 }`（默认 `{ completed: false, last_step: 0 }`），允许用户中途关窗后下次启动从上次那一步继续；`set_onboarding_step(step)` / `finish_onboarding()` 两个命令薄封装 `SettingsStore::update`，完成时关闭引导窗。
  > 步骤设计（前端 `pages/Onboarding/index.tsx`，HeroUI `Tabs(variant="underlined")` 或自绘 stepper；底部固定「上一步 / 下一步 / 跳过」）：
  >
  > 1. **欢迎**：产品介绍 + 版本号 + GitHub 链接；纯展示。
  > 2. **权限**：macOS 重点页——「辅助功能」（模拟粘贴 ⌘V）/「屏幕录制」（截图类内容预览，如未来要做）/「输入监控」（OS 级键盘钩子）。每项一行：图标 + 名称 + 状态徽章（已授权 / 未授权 / 不适用）+「打开系统设置」按钮（用 `shell::open` 跳到对应 `x-apple.systempreferences:` URL）。Rust 命令 `check_permissions() -> { accessibility, screen_recording, input_monitoring }` 用 macOS API（`AXIsProcessTrusted` 等）查询；Windows 上整页跳过或仅显示「无需额外授权」。
  > 3. **快捷键**：核心快捷键速览 + 让用户当场录入「唤出主窗」组合；复用设置面板里的快捷键录入控件。展示当前默认（macOS `Cmd+Shift+V`、Windows `Win+V`，与 8.9 联动），用户可改可跳过。
  > 4. **忽略应用**：解释「在这些 App 复制时不入库」用途（密码管理器、银行 App 等），用 `list_known_apps` 命令（已有 `apps_registry.rs` 基础）列出近期出现过的应用 + 系统常见敏感 App 预设清单（macOS：1Password、Bitwarden、Keychain Access；Windows：1Password、Bitwarden、KeePass），多选写入 `Settings.clipboard.ignored_apps`。可全跳过。
  > 5. **完成**：摘要前面选择 + 「打开主窗体验」按钮（触发 `toggle_main_window` 并 `finish_onboarding`）。
  >    设计余量：步骤数组定义在前端常量（`src/pages/Onboarding/steps.ts`），新增步骤只追加数组项 + 一个组件文件，不动外壳，方便后续补「自启」「云同步」「数据迁移」等。
  >    视觉：左侧 1/3 竖向 stepper 显示步骤名 + 当前位置高亮，右侧 2/3 是当前步骤内容；macOS 用 `vibrancy` 背景，Windows 用纯色 `bg-content1`。底部进度条按 step 计算百分比，符合现代 onboarding 直觉。
  >    i18n：`onboarding.{step1..step5}.{title,desc,action}` 与按钮文案（next / prev / skip / finish）两份语言同步补齐。
  >    触发时机：① 首次启动（`Settings.onboarding.completed == false`）；② 设置面板「关于 / 帮助」加「重新查看引导」按钮 → 调 `open_onboarding`；③ 主动重置（清掉 settings 文件）等价首次启动。不要在每次升级后强制重弹——后续大版本若新增关键步骤，单独做「版本更新提示」而非整套重走。

### 8.11 每应用规则（Per-App Rules）

- [ ] 支持按来源应用设置独立行为规则（纯文本粘贴 / 整体禁用 / 仅不捕获）
  > 目标场景：① WPS 内粘贴默认强制纯文本；② 某些游戏内 EcoPaste 全部能力关闭（不监听、不唤窗、不注入按键）；③ 某些应用仅「不捕获」复制内容，但保留手动唤出和历史粘贴能力。
  > 规则模型（Rust，`settings/model.rs`）：新增 `clipboard.app_rules: Vec<AppRule>`，`AppRule { app_id: String, app_name: String, behavior: AppBehavior }`。`app_id` 统一用 macOS bundle id / Windows exe 名（小写）做主键，`app_name` 仅展示。`AppBehavior` 三档：`force_plain_paste`、`disable_all`、`ignore_capture`。按列表顺序匹配第一条命中规则，后续可扩展优先级但本步先不做。
  > 命中时机（Rust）：
  >
  > 1. 捕获链路（watcher）在 `persist_and_notify` 前读取当前前台应用 id，命中 `ignore_capture` 或 `disable_all` 则直接 return，不入库。
  > 2. 粘贴链路（`paste_clipboard_item`）在执行前读取当前前台应用 id，命中 `force_plain_paste` 时无条件把 `plain=true`；命中 `disable_all` 时返回 `AppError::AppRuleBlocked`。
  > 3. 唤窗/快捷键链路（`toggle_window` / shortcut callback）命中 `disable_all` 时拒绝打开主窗，避免游戏内误触弹窗。
  >    平台实现：复用 7.1.5 的前台应用采集基础（macOS 走 `NSWorkspace.frontmostApplication`；Windows 增补 `GetForegroundWindow + GetWindowThreadProcessId + QueryFullProcessImageNameW` 提取 exe 名）。仅支持 macOS + Windows，不新增 Linux 分支。
  >    前端设置页：`Preference > Clipboard` 新增“应用规则”区块，列表展示「应用名 + 行为」并支持新增/删除。新增时从两路来源选择：① 最近来源应用（`clipboard_apps`）；② 当前前台应用（命令 `get_frontmost_app`）。行为文案：`force_plain_paste=纯文本粘贴`、`disable_all=在该应用中禁用 EcoPaste`、`ignore_capture=不捕获该应用复制`。
  >    与现有设置联动：`copyPlain/pastePlain` 是全局默认，`force_plain_paste` 属于按应用覆盖，优先级更高；`disable_all` 优先级最高（覆盖其他规则和全局设置）。
  >    事件与提示：命中 `disable_all` 导致操作被拒时 emit `app-rule://blocked`（payload 含 app 与行为），前端用轻提示说明“当前应用已禁用 EcoPaste”。
  >    i18n：新增 `settings.appRules.*`（标题、行为名称、空状态、添加/删除按钮、命中提示）中英文两份。

---

# 阶段 9 · 打包、签名与发布

### 9.1 图标与资源

- [x] 应用 identifier / name / version（version 取自 package.json）

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
