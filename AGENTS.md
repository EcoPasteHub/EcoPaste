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
- **组件声明统一用 `FC<Props>` + `props` 解构**，不要把解构写进函数签名：

  ```tsx
  // ✅ 正确
  interface FooProps { a: string; b?: number }
  const Foo: FC<FooProps> = (props) => {
    const { a, b = 0 } = props;
    // ...
  };

  // ❌ 不要
  const Foo = ({ a, b = 0 }: FooProps) => { ... }
  function Foo(props: FooProps) { ... }
  ```

  好处：默认值集中在函数体头部、便于调试时整体打印 `props`、便于后续 `rest` 透传。

- 状态：Valtio 只存 UI 状态和设置的本地镜像；业务数据从 Rust 命令拉取，不在前端建「数据库副本」。
- 样式：UnoCSS。
- **尺寸只走 wind4 数字制**（1 单位 = 0.25rem = 4px），写 `p-1` / `p-1.25` / `p-1.5` / `gap-2` / `rounded-2.5` / `w-36` 等数字原子类；**禁止**出现 `px` 字面量——不写 `rounded-[10px]` / `w-[144px]` / `text-[14px]` 这类任意值类，也不在 `style={{}}` 里塞数字像素（antd 组件 `tabBarStyle` 等 inline 样式同理，必要时改写 className 或在 `theme.token` 里调）。非常规尺寸通过 wind4 内置 fractional（`.25/.5/.75/.125`）拼，确实没有对应值再扩 `theme.spacing`。
- 组件优先用 Ant Design v6（`antd`），避免重复造轮子；主题切换在根节点用 `ConfigProvider` 的 `theme.algorithm` 切 `defaultAlgorithm` / `darkAlgorithm`，并把同步的 `light` / `dark` 类挂到 `<html>` 上，供 UnoCSS `dark:` 变体使用。
- antd prop 命名：`open` / `checked` / `disabled` / `onClick`（不要再用 HeroUI 的 `isOpen` / `isSelected` / `isDisabled` / `onPress`）。
- 条件 className 统一用 `cn from "@/utils/cn"`（内部 `clsx` + `tailwind-merge`，后写的同族原子类胜出）+ 对象语法 `{ "class": cond }`，不堆三元 / `&&`。**禁止**用模板字符串 / `+` 拼接 className（包括 `${className ?? ""}` 这种透传）——一律走 `cn("base...", condClass, propsClassName)`。
- **JSX 中的事件回调优先提取为命名函数**，不要在 JSX 里塞内联箭头函数 / 行内逻辑：
  - ❌ `<Button onClick={() => doSomething(x)} />` / `<Input onChange={(e) => setX(e.target.value)} />`
  - ✅ 在组件函数体内声明 `const handleClick = () => doSomething(x);`，JSX 里写 `<Button onClick={handleClick} />`
  - 同一个回调被多个元素复用时（如 `onClick` 和某快捷键 `onKeyPress`）共用同一个命名函数。
  - 命名约定：单一动作用动词（`focusSearch` / `openGithub`），通用事件处理用 `handleXxx`（`handleSubmit` / `handleChange`）。
  - 好处：JSX 只描述结构、便于打断点调试、避免每次渲染重建闭包、回调有自己的 JSDoc 说明意图。
  - **例外**：注册到 hook / API 而非 JSX prop 的一次性回调（如 `useEventListener("error", (e) => ...)`、`window.addEventListener` 回调），如果逻辑只有 2-3 行且不复用，可直接写成内联箭头函数——这类回调不在 JSX 里，没有渲染重建问题，提取成命名函数反而增加噪音。**但**：若该回调有「为什么这么做」的非显然意图（兜底、绕 bug、特殊顺序），仍需在 `useEventListener(...)` 上方写一行注释说明。
- 日志统一走 `@/utils/log`（内部委托 `tauri-plugin-log`），禁止裸 `console.*`——保证前后端日志同源进 LogDir。
- 平台 / 环境判断统一从 `@/utils/is` 引入（`isMac` / `isWin` / `isDev` 等），**禁止**在业务代码里散写 `platform() === "macos"` / `import.meta.env.DEV` 这类原始判断；新增一个判断时先去 `is.ts` 加常量，再 import 使用。
- 表达「未定义」一律用 `void 0`，**禁止**写 `undefined` 字面量（包括判等 `x === undefined`、默认值、返回值等）。理由：`undefined` 在非严格作用域里可被遮蔽，`void 0` 是表达式恒等且更短。
- 异步统一用 `async` / `await` + `try` / `catch`，**禁止**使用 `.then()` / `.catch()` / `.finally()` 链式写法（包括火并忘场景：`fn().catch(...)` 应改成 `async function wrapper() { try { await fn(); } catch {} }`）。理由：风格一致、错误处理与控制流在同一缩进、便于打断点。
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
- **守卫条件优先早返回（guard clause）**，不要把主逻辑塞进 `if` 体里：
  - ❌ `if (cond) { doMainThing(); }` —— 主流程被嵌套一层。
  - ✅ `if (!cond) return;` 空一行，然后写主逻辑。
  - 同理：多个互斥分支用「条件不满足就 return」逐个剥离，避免 `if / else if / else` 金字塔。
  - 反例：`if (isModifierPressed(e)) setActive(true);` —— 应改成 `if (!isModifierPressed(e)) return;` + `setActive(true);`。
- 函数体内**给逻辑分组留空行**，不要把多步骤压成一坨：
  - hooks 调用 / 变量声明 / 副作用 / `return` 之间至少留一空行。
  - 连续的 `await`、`Object.assign`、`if` 块等，若属于**不同语义阶段**（如「先订阅再拉首屏」「先准备数据再渲染」），用空行分隔。
  - `return` 之前留一空行——尤其是 React 组件 / hook，让早返回、JSX 与上面的逻辑视觉分离。
  - 反例：`use(x); return <Foo />;` 一行接一行；正例：中间空一行。
  - 同一组紧密相关的赋值（如解构 + 立即归一化）不强求空行，凭语义判断。
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

**Ant Design v6：**

- 官方文档（站点搜索）：https://ant.design/components/overview-cn
- 主题与 token：https://ant.design/docs/react/customize-theme-cn

**UnoCSS：**

- 官方文档：https://unocss.dev/
- presetWind4：https://unocss.dev/presets/wind4

**Tauri v2：**

- 全量文档：https://tauri.app/llms-full.txt

## 注意事项

- 旧版 `EcoPaste_bak` 仅供参考其功能与平台特化思路，**不要把它的前端直查 SQL、前端轮询监听等已废弃模式照搬过来**。
- 旧版的自定义插件（eco-window / eco-paste / eco-autostart / clipboard-x / nspanel）可评估复用，但需先确认与 Tauri v2 + 仅双平台的约定兼容。
- **`src copy/` 目录是作者废弃的草稿，AI 工具禁止读取、引用或以任何方式访问其中的文件**；该目录仅供作者本人查阅，不参与任何 AI 辅助编码流程。
