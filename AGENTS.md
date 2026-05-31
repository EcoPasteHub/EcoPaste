# AGENTS.md

> 本文件是本项目所有 AI 编码工具的**单一真相源（single source of truth）**。
> CLAUDE.md / .cursor/rules / .github/copilot-instructions.md 均引用本文件，请勿在别处重复维护规则。
> 详细的分阶段重构清单见 [TODO.md](./TODO.md)。

## 项目简介

EcoPaste 是一款跨平台剪贴板管理器，本仓库是其**重构版本**。参考实现（旧版，只读参考，**不要修改**）：

- 公开仓库：https://github.com/EcoPasteHub/EcoPaste
- 本地副本（可选加速）：`/Users/ayang/Documents/PersonalProject/2024/EcoPaste_bak` — 若已 clone 到此路径，能访问文件系统的 AI 工具优先本地读取（可直接 grep/遍历目录，快于逐个远程 fetch）。

重构目标：把核心逻辑尽量下沉到 Rust，前端只做 UI 展示与交互。

## 技术栈

| 维度        | 选型                                                   |
| ----------- | ------------------------------------------------------ |
| 桌面框架    | Tauri v2                                               |
| 前端框架    | React 19                                               |
| UI 组件库   | HeroUI v3                                              |
| 样式        | TailwindCSS v4（CSS-first，`@theme`）                  |
| Rust 数据层 | sqlx + SQLite（异步、编译期 SQL 校验、内置 migration） |
| 前端状态    | Valtio（仅 UI 状态与设置镜像）                         |
| 构建        | Vite + pnpm                                            |
| Lint/Format | Biome（前端）、rustfmt + clippy（Rust）                |

## 支持平台

**仅 macOS + Windows**。不支持 Linux。

- 不要新增任何 Linux（X11/Wayland/AppImage/deb/rpm）相关代码或依赖。
- 平台特化代码一律用 `#[cfg(target_os = "macos")]` / `#[cfg(target_os = "windows")]` 隔离。
- 优先打通 macOS（参考项目主力平台），再补 Windows。

## 核心架构原则：Rust-First

> 这是本项目最重要的约定。新增功能时**先问「这能否在 Rust 实现」**，只有 Rust 不适合时才放前端。

**必须在 Rust 实现：**

- 剪贴板监听（OS 级监听，不要在前端轮询）
- 所有数据库读写（前端**禁止**直连 SQL；旧版用 Kysely 直查的做法已废弃）
- 内容类型识别（URL / email / color / path）
- 全文搜索（SQLite FTS5）
- 历史记录清理（保留时长 / 最大条数的后台任务）
- 写回剪贴板 + 模拟粘贴
- 窗口定位计算（跟随光标 / 居中 / dock 的坐标数学）
- 图片落盘、缩略图、文件元信息读取
- 设置项持久化（由 Rust 落盘）

**保留在前端：**

- 组件渲染、虚拟滚动、瀑布流布局、动画
- 主题切换的视觉应用、CSS 变量注入
- i18n 文案渲染
- HTML 预览的 DOM 渲染与 sanitize（DOMPurify）、RTF 渲染、Markdown 渲染
- 键盘交互、列表选中态

**前后端通信：** 前端通过 `#[tauri::command]` 调用 Rust；Rust 通过 `emit` 事件通知前端刷新。事件名约定 `domain://action` 格式（`clipboard://updated` / `settings://updated` / `window://visibility` / `keyboard://nav`）；命令名 / 事件名两侧的字面量分别集中在 Rust 模块常量与 `src/constants/` 下，改名两侧同步。

## 目录结构（约定）

```
src-tauri/
  src/
    commands/    # #[tauri::command] 入口，薄封装，调用下层逻辑
    db/          # sqlx 仓储层、连接池、模型
    clipboard/   # 剪贴板读写 + OS 级监听 + 内容识别
    window/      # 窗口管理、定位计算、平台特化（NSPanel 等）
    keystroke/   # OS 级按键事件注入（当前用于模拟粘贴：macOS ⌘V / Windows Shift+Insert）。写回剪贴板在 clipboard/write.rs
    keyboard/    # OS 级键盘钩子（Windows focusable=false 主窗收不到键时，由这里 emit `keyboard://nav`）
    shortcut/    # 全局快捷键注册（tauri-plugin-global-shortcut 封装）
    tray/        # 系统托盘菜单与 i18n
    autostart/   # 开机自启（绕过 tauri-plugin-autostart 上游 bug，直接用 auto-launch crate）
    settings/    # 设置模型与持久化
    core/        # 平台 setup、应用生命周期
  migrations/    # sqlx 迁移 SQL（0001_init.sql, 0002_fts.sql ...）
src/             # 前端：components / pages / stores(valtio) / hooks / locales / utils
```

## 常用命令

```bash
pnpm install            # 安装前端依赖
pnpm tauri dev          # 开发（前端 + Rust 热重载）
pnpm tauri build        # 打包
pnpm lint               # Biome 检查
pnpm format             # Biome 格式化
cargo fmt               # Rust 格式化（在 src-tauri 下）
cargo clippy -- -D warnings   # Rust lint，警告即错误
cargo test                    # Rust 单测（在 src-tauri 下）
```

## Rust 侧约定

- 所有命令与仓储函数用 `async`，返回统一的 `Result<T, AppError>`；`AppError` 实现 `serde::Serialize` 供前端接收。
- 数据库走连接池 `SqlitePool`，存入 Tauri `State`，不要每次新建连接。
- SQL 用 `sqlx::query` / `query_as`（运行时校验），不用 `query!` 宏——避免维护 `.sqlx` 离线缓存与 `DATABASE_URL` 配置。
- migration 只增不改：已合并的迁移文件不要回头修改，新增变更写新文件（**例外**：仓库未发版前，所有改动直接合并到 `0001_init.sql`，不新增文件）。
- **改 schema 必须同步改所有相关 SQL 和测试**：新增/删除字段时，逐一检查 `SELECT` 列表、`INSERT` 列与 `bind` 参数、`UPDATE` 语句、以及 `db/*.rs` 测试里手写的结构体字面量。`sqlx::query_as` 是运行时映射，字段对不上时整个查询返回空结果而不报错——UI 上表现为"什么都不显示"。
- 表必须有 `created_at` 和 `updated_at` 两个字段，类型 `TEXT NOT NULL`；`UPDATE` 语句要同步更新 `updated_at`。
- 错误处理用 `thiserror`（定义错误类型）+ `anyhow`（内部传播），日志用 `tauri-plugin-log`。
- `commands/` 层保持薄：参数校验 + 调用 `db`/`clipboard`/`window` 等模块，不写业务逻辑。
- 处理自身写回剪贴板导致的监听回环（写回时打标记抑制下一次监听）。

## 前端侧约定

- React 19：优先用新特性（Actions、`use`、`useOptimistic`、ref as prop），不要再用 `forwardRef`。
- 状态：Valtio 只存 UI 状态和设置的本地镜像；业务数据从 Rust 命令拉取，不在前端建「数据库副本」。
- 样式：TailwindCSS v4 用 `@theme` 定义 token；优先用 HeroUI v3 组件，避免重复造轮子。
- 条件 className 统一用 `cn from "@heroui/styles"` + 对象语法 `{ "class": cond }`，不堆三元 / `&&`。
- 日志统一走 `@/utils/log`（内部委托 `tauri-plugin-log`），禁止裸 `console.*`——保证前后端日志同源进 LogDir。
- 不要在前端写 SQL、不要做内容类型识别、不要算窗口坐标——这些调用 Rust 命令。
- i18n 文案表：zh-CN（默认）/ en-US，新增文案两种语言同步补齐。
- 列表用 `react-virtuoso` 虚拟滚动；HTML 内容必须经 DOMPurify sanitize 再渲染。
- 跨端契约字符串必须走 `src/constants/` 集中维护，**禁止**在调用处写字面量。包括但不限于：
  - Tauri 命令名 → `src/constants/commands.ts`（`TAURI_COMMAND`）
  - Tauri 事件名 → `src/constants/events.ts`（`TAURI_EVENT`）
  - 后续新增的 channel 名、storage key、与 Rust 双端共享的标识符同样集中到 `src/constants/`。
  判断标准：**需要与 Rust 端字面量保持一致**或**在多个前端文件复用**的字符串就提常量；单文件局部用一次的 magic number / CSS 类名 / i18n key 不强求。

## 通用代码规范

- 函数 / 方法必须写注释：每个导出的函数、Rust 的 `pub fn`、React 组件、hook、以及非平凡的内部函数都要在声明上方写一段说明「做什么 / 关键约束」。优先用语言原生的 doc 注释：
  - TS / JS 用多行 JSDoc 块格式：
    ```ts
    /**
     * xxx
     */
    ```
    不要写成单行 `/** xxx */`。
  - Rust 用 `///`，多行就连写多行 `///`。

  Getter/setter、显然的 1 行包装、纯字面量常量可省略。

- 函数体内默认不写注释；仅当「为什么」不明显（隐藏约束、绕过特定 bug、反直觉行为）时写一行。
- 不写下列注释——发现即删：
  - 复述代码语义的废话（`// 查缓存` / `// 生成 cache_key` / `// 落盘`），well-named identifiers 本身已经说明了。
  - 引用 TODO.md 阶段号、任务编号或外部行号（"阶段 1.4"、"7.2 item 6"、"见 src-tauri/.../xxx.rs:49-50"）——重构过程中会失效。
  - 引用旧版 EcoPaste 的具体文件路径或实现（"参考旧版 `core/setup/macos.rs`"、"旧版 EcoPaste 就是这么做的"）——本仓库是重写，不背兼容包袱。
  - `// removed` / `// TODO 之前的逻辑` 之类的历史残留。
  - TS 用 `// xxx` 多行做声明上方文档（应改 JSDoc 块）、Rust 用 `// xxx` 做声明文档（应改 `///`）。
- 不写超出当前需求的抽象、兜底、向后兼容垫片。三行相似代码胜过过早抽象。
- 提交信息遵循 Conventional Commits（commitlint 校验）。
- 改动 UI 后，在浏览器/窗口里实际操作验证主路径与边界，不要只靠类型检查就声称完成。

## 外部文档参考（LLM 友好）

> 以下为官方提供的 LLM 友好文档（`llms*.txt`）。**按需取用，不要整篇抄进仓库**——遇到相关问题时再拉取对应文件。

**HeroUI v3：**

- 索引（导航入口）：https://heroui.com/react/llms.txt
- 组件 API / 用法：https://heroui.com/react/llms-components.txt
- 组合模式 / 配方：https://heroui.com/react/llms-patterns.txt

（`llms-full.txt` 为以上全量拼接，体积大、与上述重复，按需才用，不默认加载。）

**Tauri v2：**

- 全量文档：https://tauri.app/llms-full.txt

## 注意事项

- 旧版 `EcoPaste_bak` 仅供参考其功能与平台特化思路，**不要把它的前端直查 SQL、前端轮询监听等已废弃模式照搬过来**。
- 旧版的自定义插件（eco-window / eco-paste / eco-autostart / clipboard-x / nspanel）可评估复用，但需先确认与 Tauri v2 + 仅双平台的约定兼容。
