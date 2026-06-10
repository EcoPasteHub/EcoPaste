# TODO: Lightweight Mode Architecture

> 目标：为 EcoPaste 设计一套统一的 Lightweight Mode（轻量模式）架构，在保证主窗口秒开体验的前提下，降低隐藏窗口和非活跃窗口的 CPU、内存、GPU、前端渲染与 Rust 后端资源占用。
>
> 当前文件仅做架构分析与实施规划，不包含实现代码。

## 0. 设计边界

- 当前阶段只做方案设计，不修改现有源码、配置、数据库、测试、构建脚本或现有文档。
- 轻量模式不改变 EcoPaste 的 Rust-First 原则：窗口生命周期、资源调度、事件路由、后台任务控制应以 Rust 为主导，前端负责 UI 层暂停、恢复和局部状态保存。
- 仅支持 macOS 与 Windows；所有平台差异都应通过平台适配层收口，不能在业务入口散落 label 判断。
- 默认策略：
  - 主窗口 `main`：保留窗口实例，不销毁；隐藏后进入冻结 / 休眠态。
  - 非主窗口：隐藏 60 秒后默认自动销毁；再次打开时自动重建。
  - 特殊窗口：通过统一配置声明常驻、白名单、不同超时或特殊恢复逻辑。

## 1. 当前架构分析

### 1.1 Window 系统

当前窗口主要来自两类入口：

- Tauri 配置预创建窗口：`src-tauri/tauri.conf.json`
  - `main`：主剪贴板窗口，`visible: false`，`alwaysOnTop: true`，`focusable: false`，透明、无边框。
  - `preference`：偏好设置窗口，`visible: false`，居中，普通 focusable 窗口。
  - `clipboard-preview`：系统级预览 overlay，`visible: false`，`focusable: false`，初始 `1x1`。
- Rust 动态创建窗口：
  - Windows 自定义右键菜单 `context-menu` 在 `src-tauri/src/menu/context_window.rs` 里通过 `WebviewWindowBuilder` 创建，启动期创建一次并常驻隐藏。

窗口 label 常量分散在：

- Rust：`src-tauri/src/window/mod.rs`
- 前端：`src/constants/windows.ts`
- Windows context menu：`src-tauri/src/menu/context_window.rs`

当前没有一个统一的 Window Descriptor / Registry。窗口能力、创建方式、关闭策略、是否主窗口、是否 overlay、是否可销毁等都隐含在各模块逻辑中。

### 1.2 Window 创建流程

应用启动入口在 `src-tauri/src/lib.rs`：

- 注册 Tauri plugin。
- 创建并 `manage`：
  - `WindowStateStore`
  - `SettingsStore`
  - `DatabaseState`
  - 剪贴板相关 store / watcher / cleanup task
  - shortcut manager
  - autostart manager
- macOS：
  - `main` 在启动时转换为 NSPanel。
  - `clipboard-preview` 在启动时转换为不抢焦点的 NSPanel。
- Windows：
  - `context-menu` 在启动时动态创建隐藏 WebView。

因此当前大多数窗口在应用启动阶段就已经创建并加载对应前端路由。即使窗口不可见，对应 WebView、React tree、事件监听、模块级状态与部分初始化副作用仍可能存在。

### 1.3 Window 注册与管理流程

目前的窗口管理入口集中在 `src-tauri/src/window/mod.rs`：

- `show_window(app, label)`
- `hide_window(app, label)`
- `toggle_window(app, label)`
- `position_window(app, label, pos)`
- `save_all_window_states(app)`
- `hide_on_close(window)`

已有的统一能力：

- 显示前恢复窗口几何。
- 隐藏前保存窗口几何。
- 主窗口显示前按设置应用定位策略。
- 主窗口隐藏时压制并关闭 preview。
- `show_window` / `hide_window` 成功后广播 `window://visibility`。
- 应用退出时保存所有窗口几何。

不足：

- 只有 show / hide / toggle，没有生命周期阶段模型。
- `hide_on_close` 对所有窗口统一拦截 close 并 hide，导致非主窗口不会真正销毁。
- `show_window` 假设窗口已经存在；除 Windows `context-menu` 外，没有通用重建流程。
- 销毁、重建、ready handshake、idle timer、dirty guard、keepalive lease 都没有统一入口。

### 1.4 Window 销毁流程

当前基本不存在常规窗口销毁流程：

- 用户点击窗口关闭按钮时，`on_window_event` 捕获 `CloseRequested`，统一调用 `window::hide_on_close` 并 `prevent_close()`。
- 非主窗口关闭后仍保持 WebView、React、JS heap、事件监听与本地缓存。
- `context-menu` 也是启动期创建一次，后续只 hide / show。
- `clipboard-preview` 也是配置预创建，后续只 hide / show。

结论：当前架构是“预创建 + 常驻隐藏 + 复用”，不是“按需创建 + 空闲销毁 + 自动重建”。

### 1.5 Event 系统

当前事件模式：

- Rust 通过 `app.emit(...)` 广播全局事件：
  - `clipboard://updated`
  - `settings://updated`
  - `window://visibility`
  - `backup://received`
  - `keyboard://nav`
- Rust 也会对特定窗口 `window.emit(...)`：
  - `preview://updated`
  - `context-menu://show`
- 前端通过：
  - 模块级 `listen`：如 `src/stores/settings.ts` 每个 WebView 加载后订阅 `settings://updated`。
  - 组件级 `useTauriListen`：组件挂载时订阅、卸载时取消。

现状问题：

- `app.emit` 会广播给隐藏窗口；隐藏 WebView 仍可能处理事件、更新 Valtio、触发 React 重渲染或 IPC。
- 没有事件路由层区分窗口是否 visible / hidden / dormant / destroyed。
- 窗口销毁后，事件天然丢失；当前没有 pending event / replay / show-time refresh 机制。
- `window://visibility` 只有 visible 布尔值，不足以表达冻结、休眠、销毁、重建等生命周期。

### 1.6 State 管理

Rust State：

- `SettingsStore`：设置真相源，文件持久化。
- `DatabaseState`：SQLite pool，支持热替换。
- `WindowStateStore`：窗口几何持久化。
- `WatcherPause`：托盘暂停剪贴板监听。
- `ImageStore` / `AppIconStore` / `FileIconStore` / `AppsRegistry`：剪贴板资源与来源应用缓存。
- `ShortcutManager`：全局快捷键注册状态。

前端 State：

- Valtio 只作为 UI 状态与 Rust 设置镜像。
- 每个 WebView 都有自己的 JS runtime 和模块实例，因此 `settingsState`、`sourceAppsState` 等在每个窗口内是独立镜像，而不是跨 WebView 共享内存。
- 被销毁的窗口会丢失前端本地 UI 状态；恢复时需要从 Rust 重新拉取核心数据，或通过生命周期管理器提供 rehydrate payload。

## 2. 资源占用分析

### 2.1 CPU 消耗来源

Rust 常驻 CPU 来源：

- 剪贴板 watcher：
  - macOS 使用 `NSPasteboard.changeCount` 轮询，当前间隔 120ms。
  - Windows 走系统事件 `WM_CLIPBOARDUPDATE`。
- 历史清理后台任务：
  - `clipboard::cleanup::spawn` 每 60 秒 tick，一旦达到用户设置周期就清理。
- 全局快捷键：
  - 常驻注册；触发时切换窗口。
- Windows 主窗口可见期间：
  - 低级键盘钩子 `keyboard::enable_navigation_keys`。
  - 外部点击隐藏钩子 `mouse::enable_outside_click_hide`。
- 托盘菜单、autostart、backup open-file 接收等低频事件。

前端 CPU 来源：

- 所有预创建 WebView 启动后都会加载 React App、i18n、Ant Design、UnoCSS 样式和路由页。
- `main` 的 `List` 会挂载虚拟列表、订阅剪贴板更新、设置更新、窗口可见性、键盘事件，并在初始加载时发起列表查询。
- `preference` 挂载后会加载存储占用、应用元信息、来源应用列表预热。
- `clipboard-preview` 挂载后会监听 `preview://updated`，并在 preview active 时做 payload 加载、测量、动画与缓存。
- `context-menu` 在 Windows 启动创建后会挂载前端菜单页并监听事件。
- 隐藏窗口处理 `settings://updated` / `clipboard://updated` 等事件时仍可能触发状态写入与重渲染。

### 2.2 内存消耗来源

- 每个 WebView 一份 JS runtime、React tree、CSSOM、DOM、模块缓存、i18n 和 UI 库运行时。
- `main` 持有剪贴板列表数据、虚拟列表状态、卡片 DOM、pending badge 状态、preview controller refs。
- `preference` 持有设置 schema、搜索索引结果、来源应用列表、存储占用、表单状态。
- `preview` 持有最近 payload LRU cache、测量 DOM、motion 状态、图片 / 文件预览数据。
- Rust 持有 SQLite pool、设置快照、窗口状态、图片/图标 store、应用 registry cache。
- 隐藏但未销毁的非主窗口不会释放这些前端内存。

### 2.3 GPU 与渲染资源来源

- `main`、`clipboard-preview`、`context-menu` 都使用透明 / always-on-top / 无边框窗口，平台合成成本相对普通窗口更高。
- `clipboard-preview` 可扩展为全屏 overlay，active 时有 SVG 连线、backdrop blur、motion 动画、内容切换动画。
- `preference` 使用 motion 进入动画和大量 Ant Design 控件。
- Tauri hide 通常会停止可见绘制，但 WebView 进程和部分 GPU 资源不一定立即释放；销毁 WebView 才是更彻底的释放边界。

## 3. 当前存在的问题

- 隐藏窗口并不等于冻结。隐藏窗口仍保留 WebView、JS heap、事件订阅、前端状态和部分异步任务。
- 非主窗口关闭后不会释放资源，因为全局 `CloseRequested` 被拦截为 hide。
- 启动期预创建窗口导致隐藏窗口也会完成前端初始化：
  - 偏好窗口会预加载存储占用和来源应用。
  - 预览窗口会加载整套 preview 页面。
  - Windows context menu 会创建隐藏 WebView。
- 高频或中频事件没有生命周期感知：
  - `clipboard://updated` 可能抵达隐藏主窗口。
  - `settings://updated` 会广播给所有 WebView。
- 主窗口只有可见性事件，没有 dormant / resume / freeze 语义。
- 预览逻辑虽然在主窗口隐藏时有 Rust 侧压制，但前端仍需要更统一的任务暂停机制。
- 未来新增窗口时，很容易继续复制 `if label == ...` 分支，生命周期能力不能自动获得。
- 目前没有窗口销毁前的前端确认 / dirty state / keepalive lease 机制，无法安全销毁带未保存内容的窗口。
- 没有自动重建后的 ready handshake；如果直接 show 新 WebView，用户可能看到空白或未灌入设置的首帧。
- 没有统一指标确认 Lightweight Mode 的收益，后续容易只做行为改造但无法证明资源下降。

## 4. Lightweight Mode 总体设计

### 4.1 核心思路

引入 Rust 侧统一窗口生命周期管理器：`WindowLifecycleManager`。

它负责：

- 持有所有窗口的声明式描述。
- 管理窗口实例状态。
- 统一 show / hide / close / destroy / rebuild。
- 管理 idle timer 与 destroy timer。
- 管理主窗口 freeze / dormant 状态。
- 提供生命周期事件。
- 提供生命周期感知的事件路由。
- 提供特殊窗口白名单与策略覆盖。

前端配合：

- 每个 WebView 启动后注册 `window://ready`。
- 每个 WebView 监听自身生命周期事件。
- 所有订阅、定时器、RAF、动画、预览、IPC reload 通过生命周期状态做 gate。
- 非主窗口销毁前上报 dirty / keepalive 状态，避免未保存内容丢失。

### 4.2 Window Descriptor

每个窗口只在一个统一 registry 中声明一次。建议描述字段：

| 字段 | 说明 |
| --- | --- |
| label | Tauri window label |
| route | 前端 hash route |
| role | `primary` / `secondary` / `overlay` / `menu` / `system` |
| createMode | 启动预建 / 首次打开创建 / 预热创建 |
| retainPolicy | 永久保留 / 隐藏后销毁 / 永不销毁 |
| idleTimeout | 默认 60 秒，可按窗口覆盖 |
| closeBehavior | close 转 hide / close 后销毁 / close 前确认 |
| geometryPolicy | 是否保存与恢复窗口位置尺寸 |
| platformSetup | macOS NSPanel、Windows focusable/hook/topmost 等平台适配 |
| lifecycleEvents | 是否向前端发送 ready / suspend / resume / beforeDestroy |
| eventInterests | 该窗口需要接收哪些 Rust 事件 |
| keepalivePolicy | 是否允许前端持有 keepalive lease |

建议默认窗口策略：

| 窗口 | 当前来源 | 建议策略 |
| --- | --- | --- |
| `main` | Tauri 配置预创建 | 永久保留；hidden 后进入 dormant；不销毁 |
| `preference` | Tauri 配置预创建 | 改为按需创建；隐藏 60 秒后销毁；打开自动重建 |
| `clipboard-preview` | Tauri 配置预创建 | 支持按需创建或预热；隐藏 60 秒后销毁；active 期间禁止销毁 |
| `context-menu` | Windows Rust 动态创建 | 支持按需创建；隐藏 60 秒后销毁；也可按 descriptor 缩短超时 |
| 未来编辑窗口 | 待新增 | 默认隐藏 60 秒销毁；dirty 时保留 |
| 未来图片 / 文件预览 | 待新增 | 默认隐藏 60 秒销毁；大资源释放优先级高 |
| 未来系统级窗口 | 待新增 | 通过 descriptor 标记 permanent / exempt |

### 4.3 生命周期事件

在现有 `window://visibility` 之外新增更完整的生命周期事件。建议保持 `domain://action` 风格：

- `window://created`
- `window://ready`
- `window://shown`
- `window://hidden`
- `window://suspended`
- `window://resumed`
- `window://before-destroy`
- `window://destroyed`

事件 payload 至少包含：

- `label`
- `phase`
- `generation`
- `reason`
- `visible`
- `deadlineMs`（用于 before-destroy）

`generation` 用于避免销毁计时器、异步创建、show 请求之间发生竞态。

### 4.4 事件路由设计

新增生命周期感知的 Event Router：

- 低频、必须一致的事件：
  - `settings://updated` 可以继续广播，或在窗口重建时通过首屏 `get_settings` 补齐。
- 高频、可合并的事件：
  - `clipboard://updated` 对 dormant 主窗口不立即 reload，只记录 pending refresh / badge。
  - 对已销毁窗口不发送；重建后由该窗口首屏查询补齐。
- 定向事件：
  - `preview://updated` 只发送给 preview 窗口。
  - `context-menu://show` 只发送给 context-menu 窗口，并应在窗口 ready 后发送。
- 窗口生命周期事件：
  - 尽量定向到目标窗口，同时 Rust 内部状态也要更新，避免目标窗口已销毁导致状态丢失。

原则：事件不应该成为隐藏窗口持续工作的入口。

### 4.5 前端 Lifecycle Boundary

前端新增统一生命周期层：

- 全局 `windowLifecycleState`：记录当前 WebView 的 `label`、`phase`、`visible`、`dormant`、`generation`。
- `useWindowLifecycle`：读取当前窗口生命周期。
- `useLifecycleTauriListen`：在 suspended / dormant 时自动忽略或缓冲非必要事件。
- `useLifecycleTask`：统一管理 timer、RAF、异步请求取消。
- `WindowFreezeBoundary`：主窗口 dormant 时暂停非必要 UI 更新。
- `beforeDestroy` 处理器：允许页面保存临时 UI 状态、释放大对象、拒绝销毁或申请 keepalive。

## 5. Window 生命周期设计

### 5.1 生命周期阶段

统一状态机：

| Phase | 含义 |
| --- | --- |
| `notCreated` | descriptor 已注册，但 Tauri WebView 不存在 |
| `creating` | 正在创建窗口 |
| `created` | WebView 已创建，但前端未 ready |
| `ready` | 前端已完成基础初始化，可安全 show / emit payload |
| `visible` | 窗口可见 |
| `hiddenWarm` | 刚隐藏，保留实例，允许快速恢复 |
| `dormant` | 主窗口专用：实例保留，但前端与后端进入冻结策略 |
| `destroyPending` | 非主窗口隐藏超时，正在通知前端即将销毁 |
| `destroying` | 正在关闭 WebView，释放资源 |
| `destroyed` | WebView 已销毁，仅保留 descriptor 与持久状态 |

### 5.2 主窗口生命周期

主窗口不进入 `destroyPending` / `destroyed`。

建议路径：

- 启动：`created` -> `ready` -> `hiddenWarm`
- 打开：`hiddenWarm` / `dormant` -> `visible`
- 隐藏：`visible` -> `hiddenWarm`
- 隐藏超过短暂宽限期：`hiddenWarm` -> `dormant`
- 再次打开：`dormant` -> `visible`

隐藏宽限期建议 2 到 5 秒，用于避免用户快速打开 / 关闭时频繁 suspend / resume。

### 5.3 非主窗口生命周期

建议路径：

- 首次打开：`notCreated` -> `creating` -> `created` -> `ready` -> `visible`
- 隐藏 / 关闭：`visible` -> `hiddenWarm`
- 隐藏超过 60 秒：`hiddenWarm` -> `destroyPending` -> `destroying` -> `destroyed`
- 再次打开：`destroyed` -> `creating` -> `created` -> `ready` -> `visible`

销毁前必须：

- 保存窗口几何。
- 取消 destroy timer 的 stale generation。
- 通知前端 `before-destroy`。
- 检查 dirty / keepalive lease。
- 释放与该窗口绑定的 Rust 后端任务。
- 关闭 WebView 时绕过全局 hide-on-close 拦截。

### 5.4 CloseRequested 处理

当前 `CloseRequested` 全局拦截为 hide。轻量模式下应改为：

- 查询 `WindowLifecycleManager` 的 descriptor。
- 主窗口：继续 hide，不销毁。
- destroy-after-idle 窗口：关闭按钮触发 hide，并启动 idle timer。
- immediate-destroy 窗口：可直接进入 destroyPending。
- dirty 窗口：阻止关闭并请求前端确认，或保留到 keepalive release。
- internal destroy：生命周期管理器设置内部关闭许可，避免 close 再被 hide-on-close 拦截。

## 6. 主窗口冻结方案

### 6.1 冻结目标

主窗口隐藏后：

- 保留 WebView 与 React 状态，保证秒开。
- 不做实时渲染、动画、预览、轮询、列表 reload。
- 不处理隐藏状态下无意义的 UI 事件。
- 只保留重新打开所需的最小状态。

### 6.2 Rust 侧冻结

已有能力：

- Windows 主窗口隐藏时禁用键盘钩子和外部点击钩子。
- 主窗口隐藏时关闭并压制 preview。
- 主窗口显隐广播 `window://visibility`。

需要新增：

- 主窗口隐藏进入 `hiddenWarm`，宽限期后进入 `dormant`。
- dormant 时 Event Router 不向主窗口发送高频 UI 刷新事件，或只发送可合并的 pending 标记。
- dormant 时拒绝 preview show 请求。
- dormant 时关闭 context menu。
- dormant 时保留剪贴板 watcher、DB、托盘、全局快捷键等真正后台能力。

不能暂停：

- 剪贴板监听与入库：这是应用核心后台能力。
- 全局快捷键：用户需要唤起窗口。
- 托盘菜单。
- 设置与数据库 state。
- 历史清理任务，除非未来提供单独节能策略。

可以延迟或合并：

- 主窗口列表 reload。
- 新内容 badge 计算。
- 预览 payload 查询。
- 文件 / 图片预览解析。
- UI 相关事件。

### 6.3 前端冻结

主窗口 dormant 时：

- `List` 不立即响应 `clipboard://updated` 做 reload，只记录 pending refresh。
- `useClipboardPreviewController` 取消 hover timer、hide timer、RAF、keyboard preview frame。
- 搜索防抖任务取消或不向查询层提交。
- 关闭 NoteModal / context menu / preview 这类依赖可见交互的浮层，或保存其状态后隐藏。
- Virtuoso 不执行 scrollToIndex、loadMore、measure 等动作。
- motion / transition 不启动新的动画。
- `useKeyboardEvent` 只在 visible 时处理 UI 快捷键。
- `settings://updated` 可以更新镜像，但不触发隐藏窗口重型数据 reload。

主窗口 resume 时：

- 根据 pending refresh 决定是否 reload。
- 根据设置恢复分组、滚动和默认焦点。
- 恢复键盘导航、hover preview、菜单操作。
- 重新校验当前选中项是否仍存在。

### 6.4 Dormant 分级

建议主窗口支持三级隐藏状态：

| 等级 | 触发 | 行为 |
| --- | --- | --- |
| `hiddenWarm` | 刚隐藏 | 保留订阅，但事件 handler 轻量早退；便于快速恢复 |
| `dormant` | 隐藏超过短暂宽限期 | 暂停非必要订阅、取消 timer / RAF、合并事件 |
| `resumePending` | 用户打开窗口到 ready | 恢复事件处理，必要时补拉数据 |

## 7. 自动销毁方案

### 7.1 默认策略

非主窗口默认：

- hide / close 后进入 `hiddenWarm`。
- 启动 60 秒 idle timer。
- 期间再次 show：取消 timer，恢复 visible。
- 60 秒后进入 `destroyPending`。
- 若无 dirty / keepalive：销毁 WebView。
- 若存在 dirty / keepalive：延迟销毁并定期重试，或等待 release。

### 7.2 销毁安全条件

窗口可以销毁的条件：

- 当前不可见。
- 当前 generation 仍匹配 timer 创建时的 generation。
- 没有 active keepalive lease。
- 没有 dirty state。
- 没有活跃系统对话框、文件选择器、备份导入、快捷键录制等必须完成的流程。
- 没有绑定该窗口的进行中 Rust 后台任务，或任务已取消 / 可安全分离。

### 7.3 前端 before-destroy 协议

非主窗口销毁前，Rust 发送 `window://before-destroy`。

前端处理：

- 保存需要恢复的轻量 UI 状态。
- 释放大对象引用和 cache。
- 取消 timer / RAF / async request。
- 如果存在未保存内容，申请 keepalive 或返回 veto。

Rust 侧建议有短超时：

- 常规窗口给 200 到 500ms。
- 有明确 dirty / keepalive 的窗口不销毁。
- 超时无响应时，如果窗口不是 dirty protected，可继续销毁。

### 7.4 Keepalive Lease

前端可通过命令申请窗口保活：

- 快捷键录制期间。
- 备份导入 / 导出进行中。
- 编辑窗口有未保存内容。
- 文件选择器或系统对话框进行中。
- preview 正在加载大资源且即将显示。

lease 需要：

- label
- reason
- owner id
- timeout
- release

超时是兜底，避免异常路径永久保活。

### 7.5 后端资源释放

WebView 销毁自然释放：

- React tree。
- JS heap。
- DOM。
- WebView 事件监听。
- 前端 cache。
- GPU surface。

Rust 仍需主动管理：

- 窗口几何保存。
- preview state 清空。
- 与窗口绑定的 pending task cancel token。
- context menu pending payload 清空。
- per-window event queue 清空或转为 show-time pending。

不应释放：

- 全局 DB pool。
- SettingsStore。
- Clipboard watcher。
- Tray。
- Global shortcuts。
- 图片 / 图标 store。

## 8. 自动重建方案

### 8.1 重建入口

所有打开窗口的入口都应走统一 `open/show window` 流程：

- 全局快捷键。
- 托盘菜单。
- 前端命令。
- macOS reopen。
- backup open-file。
- preview hover / keyboard 请求。
- Windows context menu 请求。

流程：

- 查询 descriptor。
- 如果窗口存在且 ready：取消 destroy timer，恢复几何，show。
- 如果窗口不存在：按 descriptor 创建。
- 执行平台 setup。
- 等待前端 `window://ready` 或短超时。
- 发送 pending payload。
- show。
- 发送 `window://shown`。

### 8.2 Ready Handshake

重建 WebView 后，不能假设前端已可接收事件。

建议：

- 前端 App 在 `settingsReady` 完成、router mount 后发送 `window://ready`。
- payload 包含 label、route、generation。
- Rust manager 在 ready 后再发送：
  - `preview://updated`
  - `context-menu://show`
  - `backup://received`
  - rehydrate payload
- 若超时未 ready：
  - 可先 show 窗口，但需要保留 pending payload，等待 ready 后补发。

### 8.3 状态恢复

必须恢复：

- 窗口几何。
- 设置快照。
- 当前数据库数据。
- 必需的 pending event。

可选恢复：

- preference active tab / section / search query。
- 编辑窗口 draft。
- preview 上一次目标。
- context menu payload。

建议原则：

- 核心业务状态放 Rust 或数据库。
- 非主窗口 UI 暂态默认可丢弃。
- 用户可感知的未保存输入必须通过 dirty / draft / keepalive 保护。

### 8.4 避免用户感知差异

为避免自动销毁后首次打开出现明显空白：

- 对常用窗口可在用户触发前做低优先级预热。
- show 前等待 ready，或用平台窗口隐藏加载后再显示。
- preference 这类窗口将重型数据加载改为打开后渐进式展示。
- preview 窗口可在主窗口 visible 且用户启用 preview 时预热，但仍允许 idle destroy。

## 9. 未来扩展方案

### 9.1 新窗口接入规则

未来新增窗口时，只允许通过统一 descriptor 接入生命周期能力：

- 添加 label。
- 添加 route。
- 选择 role。
- 选择 retain policy。
- 声明 idle timeout。
- 声明是否允许 dirty / keepalive。
- 声明事件兴趣。
- 声明平台 setup。

不应在 show / hide 主流程继续添加 label-specific 分支。

### 9.2 窗口类型模板

建议提供几类模板：

| 模板 | 默认行为 |
| --- | --- |
| `PrimaryRetained` | 主窗口；永不销毁；隐藏后 dormant |
| `SecondaryEphemeral` | 设置、偏好、编辑；隐藏 60 秒销毁 |
| `PreviewOverlayEphemeral` | 图片 / 文件 / 剪贴板预览；隐藏 60 秒销毁，可预热 |
| `MenuEphemeral` | 右键菜单 / 工具菜单；隐藏后可较快销毁 |
| `SystemPersistent` | 必须常驻的系统窗口；默认豁免销毁 |

### 9.3 特殊窗口白名单

未来可能需要常驻窗口：

- 系统级辅助窗口。
- 全局输入捕获窗口。
- 托盘或平台 workaround 窗口。
- 需要极低延迟的 overlay。

白名单必须是 descriptor 配置，不应是业务代码里的 label 判断。

可配置项：

- `retainPolicy: permanent`
- `idleTimeout: none`
- `suspendWhenHidden: true/false`
- `receivesBackgroundEvents: true/false`
- `reason: systemRequirement`

### 9.4 事件兴趣声明

每个窗口声明自己需要哪些事件：

- `main`：
  - visible 时接收 `clipboard://updated`。
  - dormant 时只接收 pending summary 或打开时补拉。
  - 始终接收必要设置变化。
- `preference`：
  - visible 时接收 `backup://received`、`settings://updated`。
  - destroyed 时由 open-time payload 补发备份导入事件。
- `preview`：
  - 只接收 active preview payload。
- `context-menu`：
  - 只接收 show payload。

这能避免隐藏或销毁窗口继续处理无意义事件。

## 10. 风险分析

### 10.1 Tauri 生命周期风险

- 当前全局 `CloseRequested` 会拦截所有 close；内部销毁必须有 bypass 标记，否则销毁会变成 hide。
- 不同平台 WebView 销毁后的句柄行为不同，需要统一处理 missing window。
- Tauri 配置预创建窗口迁移为动态创建时，bundle capability、窗口权限、URL、透明、focusable、alwaysOnTop 等选项必须完整复刻。
- Dev 模式 HMR 与多窗口动态创建可能有额外边界。

### 10.2 macOS 风险

- `main` 与 `clipboard-preview` 使用 NSPanel；销毁后重建必须重新执行 panel 转换和 event handler 绑定。
- NSPanel handler 重复绑定可能造成事件重复或内存泄漏。
- 主窗口 show 有 16ms 延迟，生命周期事件必须与真实 panel show 时机一致。
- Dock reopen 行为需要与 preference 自动重建兼容。

### 10.3 Windows 风险

- 主窗口 `focusable=false` 依赖低级键盘钩子和外部点击钩子；冻结 / resume 必须保持当前可见时启用、隐藏时禁用的语义。
- context-menu 当前用隐藏 WebView 替代原生菜单；自动销毁后首次右键必须创建、ready、定位、发送 payload，不能明显延迟。
- preview topmost 与 full-screen overlay 需要在重建后重新设置。

### 10.4 事件丢失风险

- 窗口销毁期间发送给它的事件会丢失。
- backup open-file 需要确保 preference 重建后仍能收到 payload。
- settings 更新在窗口销毁期间不需要 replay，但窗口重建必须拉取最新快照。
- clipboard 更新可合并，但必须保证主窗口打开后列表准确。

### 10.5 用户状态风险

- 偏好页搜索、滚动、tab 状态销毁后可能丢失。
- 编辑窗口未保存内容不能被销毁。
- 快捷键录制、备份导入、文件选择器期间销毁会破坏流程。
- 自动销毁时间过短会让用户感知到重建延迟。

### 10.6 性能收益风险

- 如果只销毁窗口但仍广播事件、仍启动重型 Rust 后台任务，收益有限。
- 如果主窗口冻结过度，打开时需要大量补拉，反而破坏秒开体验。
- 如果所有非主窗口都保留复杂 rehydrate 状态，架构复杂度会上升。

### 10.7 数据与兼容风险

- 轻量模式本身不需要数据库结构变更。
- 当前项目版本仍是 `0.6.0-beta.3`，未来如果实现时涉及设置结构调整，可按未发版阶段策略直接调整；但真正发版后需考虑设置兼容。
- 轻量模式是用户可感知能力，真正实现时应同步更新 CHANGELOG。

## 11. 实施路线图

### Phase 1: 生命周期基础设施，不改变销毁行为

目标：先建立统一模型和观测，不急于销毁窗口。

- [ ] 建立 Rust `WindowLifecycleManager` 与 Window Descriptor registry。
- [ ] 把现有 label、route、窗口策略集中到 registry。
- [ ] 将 show / hide / toggle / close requested 路径接入 manager。
- [ ] 保持现有行为：所有窗口仍 hide，不自动 destroy。
- [ ] 新增生命周期 phase 与 generation。
- [ ] 保留兼容 `window://visibility`，同时新增更细生命周期事件。
- [ ] 前端新增 `useWindowLifecycle` 与生命周期 store。
- [ ] 每个 WebView 启动后发送 ready 事件。
- [ ] 为窗口状态、事件发送、隐藏时长、重建耗时增加日志和调试指标。
- [ ] 梳理并记录每个窗口的事件兴趣和可暂停任务。

验收标准：

- 行为与当前版本一致。
- 日志能看见窗口 phase 转换。
- 所有窗口 show / hide 仍正常。
- 没有新增窗口 hard-coded 分支。

### Phase 2: 主窗口冻结与生命周期感知事件

目标：主窗口不销毁，但隐藏后接近冻结。

- [ ] 主窗口隐藏后进入 `hiddenWarm`，宽限期后进入 `dormant`。
- [ ] `clipboard://updated` 对 dormant 主窗口改为合并 pending，不立即 reload。
- [ ] 主窗口 resume 时按 pending 状态补拉列表。
- [ ] preview controller 在 hidden / dormant 时取消 timer、RAF、hover、keyboard preview。
- [ ] List、Header、SearchInput、keyboard handler 接入生命周期 gate。
- [ ] 预览窗口 show 请求在主窗口 dormant / hidden 时直接拒绝或 no-op。
- [ ] motion / animation / measurement 在 dormant 时不触发。
- [ ] 保留核心后台任务：剪贴板 watcher、global shortcuts、tray、cleanup、DB。

验收标准：

- 主窗口仍能秒开。
- 隐藏期间复制内容不会触发主窗口列表反复 reload。
- 再次打开主窗口后数据准确。
- 预览不会在主窗口隐藏后残留或被过期请求唤起。

### Phase 3: 非主窗口自动销毁与自动重建

目标：非主窗口隐藏 60 秒后释放 WebView，打开时自动重建。

- [ ] 将 preference 从“启动期必须可用”改为按需创建或可销毁。
- [ ] 为非主窗口实现 idle timer、destroyPending、beforeDestroy、destroyed。
- [ ] 实现 internal close bypass，避免销毁被 hide-on-close 拦截。
- [ ] 实现 dirty / keepalive lease 协议。
- [ ] 实现 destroyed 窗口的 show-time rebuild。
- [ ] 实现 ready handshake 后再发送 pending payload。
- [ ] preference 支持重建后恢复必要状态和最新设置。
- [ ] Windows context-menu 支持销毁后首次右键自动重建、ready、定位、show payload。
- [ ] clipboard-preview 支持销毁后自动重建，并重做 macOS NSPanel / Windows topmost setup。
- [ ] 为新增窗口提供 descriptor 模板与接入清单。

验收标准：

- preference 隐藏 60 秒后 WebView 被销毁，再次打开可自动重建。
- context-menu / preview 不因销毁出现空白、错位、事件丢失。
- 有 dirty / keepalive 的窗口不会被误销毁。
- 非主窗口销毁后内存和 GPU 占用可观察下降。

### 后续优化

- [ ] 增加用户可配置的轻量模式开关与 idle timeout。
- [ ] 增加开发调试面板显示窗口 phase、generation、lastActive、lease。
- [ ] 对主窗口 dormant 后的资源占用做基准测试。
- [ ] 对偏好页、preview、context-menu 的销毁收益分别做对比。
- [ ] 若未来新增大量窗口，考虑由 registry 生成前后端窗口常量，减少 label 漂移。
