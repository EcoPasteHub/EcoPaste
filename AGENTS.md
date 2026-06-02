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

| 维度        | 选型                                                    |
| ----------- | ------------------------------------------------------- |
| 桌面框架    | Tauri v2                                                |
| 前端框架    | React 19                                                |
| UI 组件库   | Ant Design v6                                           |
| 样式        | UnoCSS（`presetWind4`，原子化 + 兼容 Tailwind v4 写法） |
| Rust 数据层 | sqlx + SQLite（异步、编译期 SQL 校验、内置 migration）  |
| 前端状态    | Valtio（仅 UI 状态与设置镜像）                          |
| 构建        | Vite + pnpm                                             |
| Lint/Format | Biome（前端）、rustfmt + clippy（Rust）                 |

## 支持平台

**仅 macOS + Windows**。不支持 Linux。

- 不要新增任何 Linux（X11/Wayland/AppImage/deb/rpm）相关代码或依赖。
- 平台特化代码一律用 `#[cfg(target_os = "macos")]` / `#[cfg(target_os = "windows")]` 隔离。
- macOS 与 Windows **同步开发**：新增功能需同时给出两端实现（或显式标注 TODO），不要长期只在单平台可用。

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
    commands/    # #[tauri::command] 入口
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
- `AppError` 序列化为 `{ kind, message }`。`message` 是**给用户看的根因文案**，**禁止**叠加 `"xxx failed: {err}"` 这种动作前缀（动作上下文由前端 `commands/index.ts` 的 `label` 拼 toast）。需要额外上下文走 `log::error!` / `log::warn!`，不塞进 `message`。
- `commands/` 层保持薄：参数校验 + 调用下层模块，不写业务逻辑。
- 处理自身写回剪贴板导致的监听回环（写回时打标记抑制下一次监听）。

## 前端侧约定

- React 19：优先用新特性（Actions、`use`、`useOptimistic`、ref as prop），不要再用 `forwardRef`。
- **组件声明统一用 `FC<Props>` + 函数体内 `props` 解构**，不要把解构写进函数签名：

  ```tsx
  const Foo: FC<FooProps> = (props) => {
    const { a, b = 0 } = props;
  };
  ```

- **解构时优先用 `...rest` 收尾**：若剩余字段需要整体透传或展开，用 `...rest` 而非逐一列举；只用几个字段且无需透传则按需解构。

- 状态：Valtio 只存 UI 状态和设置的本地镜像；业务数据从 Rust 命令拉取，不在前端建「数据库副本」。
- 样式：UnoCSS。
- **颜色只用 antd Design Token 映射的语义色类名**（由 `presetAntdColors` 生成，如 `text-text-primary` / `bg-container` / `border-border`，完整列表见 `src/unocss/presetAntdColors.ts`）。**禁止**任意色值：不写 `text-[#333]` / `border-red-500` 等 hex / rgb / Tailwind 原生色，也不在 `style={{}}` 里塞 `color` / `background`。需要新颜色先扩 preset，不要绕过。
- **尺寸只走 wind4 数字制**（1 单位 = 0.25rem = 4px），如 `p-1.5` / `gap-2` / `rounded-2.5` / `w-36`。**禁止**`px` 字面量：不写 `w-[144px]` 这类任意值类，也不在 `style={{}}` 里塞数字像素（antd `tabBarStyle` 等 inline 样式同理，必要时改 className 或调 `theme.token`）。非常规尺寸用内置 fractional（`.25/.5/.75/.125`）拼，没有再扩 `theme.spacing`。
- 组件优先用 Ant Design v6（`antd`），避免重复造轮子；主题切换在根节点用 `ConfigProvider` 的 `theme.algorithm` 切 `defaultAlgorithm` / `darkAlgorithm`，并把同步的 `light` / `dark` 类挂到 `<html>` 上，供 UnoCSS `dark:` 变体使用。
- antd prop 命名：`open` / `checked` / `disabled` / `onClick`（不要再用 HeroUI 的 `isOpen` / `isSelected` / `isDisabled` / `onPress`）。
- 条件 className 统一用 `cn from "@/utils/cn"`（内部 `clsx` + `tailwind-merge`，后写的同族原子类胜出）+ 对象语法 `{ "class": cond }`，不堆三元 / `&&`。**禁止**用模板字符串 / `+` 拼接 className（包括 `${className ?? ""}` 这种透传）——一律走 `cn("base...", condClass, propsClassName)`。
- **JSX 中的事件回调优先提取为命名函数**，不在 JSX 里塞内联箭头函数 / 行内逻辑：
  - ❌ `<Button onClick={() => doSomething(x)} />`
  - ✅ 函数体内 `const handleClick = () => { doSomething(x); };` + `<Button onClick={handleClick} />`
  - 命名约定：单一动作用动词（`focusSearch`），通用事件处理用 `handleXxx`。
  - **例外**：注册到 hook / API 而非 JSX prop 的一次性回调（如 `useEventListener("error", ...)`、`window.addEventListener` 回调），逻辑 2-3 行且不复用时可内联。但若有非显然意图（兜底、绕 bug、特殊顺序），需在上方加一行注释说明。
- 日志统一走 `@/utils/log`（内部委托 `tauri-plugin-log`），禁止裸 `console.*`——保证前后端日志同源进 LogDir。
- 平台 / 环境判断统一从 `@/utils/is` 引入（`isMac` / `isWin` / `isDev` 等），**禁止**在业务代码里散写 `platform() === "macos"` / `import.meta.env.DEV` 这类原始判断；新增一个判断时先去 `is.ts` 加常量，再 import 使用。
- 取当前窗口一律用 `getCurrentWebviewWindow()`（`@tauri-apps/api/webviewWindow`），**禁止**用 `getCurrentWindow()`（`@tauri-apps/api/window`）。前者返回 `WebviewWindow`，既能调窗口方法又能调 webview 方法（如 `setZoom`、emit 到自身 webview），后者只有窗口能力，遇到要操作 webview 时还得二次拿实例。
- 涉及 UI 的快捷键处理必须区分平台/窗口：**Windows 主窗口** 场景一律走 Rust `keyboard/` 的系统键盘钩子（Rust emit `keyboard://nav` 等事件），不要只依赖 Web `keydown`；其它场景默认走 Web 键盘事件即可。
- 表达「未定义」一律用 `void 0`，**禁止** `undefined` 字面量（`undefined` 在非严格作用域里可被遮蔽）。
- 异步统一用 `async` / `await` + `try` / `catch`，**禁止** `.then()` / `.catch()` / `.finally()` 链式写法（火并忘场景也用 `async` 包装 + `try/catch`）。
- **箭头函数函数体一律 `{}` + 显式 `return`**，禁止单表达式隐式返回：
  - ❌ `arr.map((x) => ({ id: x.id }))` / `const fn = (x) => x + 1;`
  - ✅ `arr.map((x) => { return { id: x.id }; })` / `const fn = (x) => { return x + 1; };`
  - 便于打断点、加临时变量 / 日志时不用回头补 `{}`。**例外**：JSX 内联回调本就禁止（见上）。
- **`useEffect` 回调禁止直接声明为 `async`**——React 不处理其返回的 Promise，cleanup 会丢失。需要异步初始化时改用 `useMount` + `useUnmount`（ahooks）；清理句柄（如 `unlisten`）用 `useRef` 跨两个 hook 传递。`useEffect` 只用于**纯同步**副作用或依赖项驱动的场景。
- i18n 文案表：zh-CN（默认）/ en-US，新增文案两种语言同步补齐。
- 列表用 `react-virtuoso` 虚拟滚动；HTML 内容必须经 DOMPurify sanitize 再渲染。
- **跨端契约字符串集中到 `src/constants/`**，禁止在调用处写字面量。已有：Tauri 命令名 → `commands.ts`（`TAURI_COMMAND`）、Tauri 事件名 → `events.ts`（`TAURI_EVENT`）；后续 channel 名 / storage key / 与 Rust 共享的标识符同样集中。判断标准：**需要与 Rust 端字面量一致**或**多个前端文件复用**就提常量；单文件一次性的 magic number / CSS 类名 / i18n key 不强求。

## 通用代码规范

- 函数 / 方法必须在声明上方写文档注释，说明「做什么 / 关键约束」：
  - TS / JS 用多行 JSDoc（`/**` 独立一行 + `*/` 独立一行），不要单行 `/** xxx */`。
  - Rust 用 `///`，多行连写多行 `///`。
  - getter/setter、显然的 1 行包装、纯字面量常量可省。

- 函数体内默认不写注释；仅当「为什么」不明显（隐藏约束、绕过特定 bug、反直觉行为）时写一行。
- **守卫条件优先早返回**：`if (!cond) return;` 然后写主逻辑，不要把主流程塞进 `if` 体里嵌套一层。多个互斥分支同理，避免 `if / else if / else` 金字塔。
- 函数体内**给逻辑分组留空行**：hooks / 变量声明 / 副作用 / `return` 之间至少一空行；不同语义阶段的连续 `await` / `if` 块也用空行分隔。`return` 之前留一空行（尤其 React 组件 / hook）。
- 不写下列注释：
  - 复述代码语义的废话（`// 查缓存` / `// 落盘`）。
  - 引用 TODO.md 阶段号、任务编号或外部行号（重构中会失效）。
  - 引用旧版 EcoPaste 的具体路径或实现（本仓库不背兼容包袱）。
  - `// removed` / `// TODO 之前的逻辑` 之类历史残留。
- 不写超出当前需求的抽象、兜底、向后兼容垫片。三行相似代码胜过过早抽象。
- **不过度设计 React hook / 工具函数**——React 19 能用编译器自动 memo 就别手动包：
  - 不为「调用方可能传内联函数」预先 `useMemoizedFn` / `useRef` 锁存。
  - 不为「可能传对象 / 数组」加 `useRef` 缓存 + 自比对——直接把入参类型约束成基础类型。
  - 能用 `useSnapshot` + `useEffect` 表达就不要手写 `subscribe` + `useMount` + `useUnmount` 重造。
  - 先写最短能跑的版本，遇到真问题再加抽象。
- 提交信息遵循 Conventional Commits（commitlint 校验）。
- 改动 UI 后，在浏览器/窗口里实际操作验证主路径与边界，不要只靠类型检查就声称完成。

## 外部文档参考（LLM 友好）

> 官方 `llms*.txt`，**按需取用**，不要整篇抄进仓库。

- Ant Design v6：https://ant.design/components/overview-cn · [主题与 token](https://ant.design/docs/react/customize-theme-cn)
- UnoCSS：https://unocss.dev/ · [presetWind4](https://unocss.dev/presets/wind4)
- Tauri v2：https://tauri.app/llms-full.txt
