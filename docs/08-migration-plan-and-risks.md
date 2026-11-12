# 迁移顺序与风险

## 1. 当前阶段声明

本文件只给出后续迁移的候选顺序和验证门槛，不代表已经进入重构。

当前文档分支 `refactor/go-nsgo-docs` 的交付物只有 `docs/` 下的分析文档。未完成或未授权以下工作：

- 创建 Go 业务代码。
- 创建 AppKit UI。
- 修改 Rust/Tauri/React 业务逻辑。
- 修改数据库 schema 或 migration。
- 更换 SQLite 驱动。
- 迁移模块或发布版本。

## 2. 推荐实施顺序

### Phase 0：技术 spike

目标：先证伪最大风险，不写产品代码。

1. nsgo `core.Native` + `NSSlider` / `NSButton` / `NSTextField`。
2. `NSScrollView + NSTableView` 最小列表。
3. nsgo Panel / 自建 NSPanel：show without activating、can become key、hide on blur。
4. 剪贴板读 plain/html/rtf/image/files，写回 plain/html/rtf/image/files。
5. 来源应用识别 + changeCount 监听 + 写回回环抑制。
6. CGEvent ⌘V + Accessibility 权限。
7. 候选 SQLite 驱动打开真实旧库并执行 FTS5 trigram。

退出条件：

- 每个 spike 有可重复的验证记录。
- 明确 purego/objc 是否足以支撑 NSTableView delegate。
- 明确 SQLite 驱动选择。
- 明确 nsgo panel 是否适合作为主窗口壳。

### Phase 1：数据兼容层

目标：在不改变产品 UI 的情况下证明 Go 能安全读写旧数据。

- 数据根解析与自定义目录 manifest。
- 只读 schema 探测。
- Go schema ledger。
- 旧库读取、FTS5 查询、LIKE 降级。
- 图片/图标资源解析。
- settings JSON 兼容。
- 旧版 history 导入。
- `.ecopastebak` 明文和加密格式交叉验证。

退出条件：

- 使用真实旧库运行查询和列表构造，结果与 Rust 版本逐项对比。
- 任何写入前有可恢复备份。
- 加密备份与 Rust 生成的样本互通。

### Phase 2：无 UI 的剪贴板核心

目标：先证明采集与持久化正确，再接 UI。

- watcher。
- payload 读取和类型优先级。
- 文本/图片/文件建模。
- 敏感检测。
- 去重。
- app 记录和图标。
- 写回和 guard。
- 历史清理。

退出条件：

- 复制文本、HTML、RTF、截图、多文件、图片文件后记录正确。
- 自身写回不产生新记录。
- 连续相同内容只累加 `use_count`。
- 敏感内容和排除应用规则与 Rust 版本一致。

### Phase 3：AppKit 主窗口

目标：达到可日常使用的列表和搜索体验。

- NSPanel/Window。
- 单 Native 根。
- 搜索框、过滤、列表、分页。
- 列表 cell：文本、图片、文件、图标。
- 选中、键盘导航、双击/Enter 粘贴、Space 预览。
- 右键菜单。
- 收藏/置顶/备注/分组。
- 空态和错误态。

退出条件：

- 1000+ 条旧数据滚动和搜索流畅。
- 置顶/收藏/分组与旧库一致。
- 鼠标、键盘、菜单、拖出主路径可用。
- 不出现 WebView。

### Phase 4：系统集成

目标：恢复 macOS 产品级行为。

- NSPanel pin/auto-hide。
- 窗口位置和生命周期。
- 快捷键和冲突。
- 托盘菜单。
- 粘贴时序。
- 拖出。
- 权限引导。
- 单实例、Dock reopen、文件打开。

退出条件：

- 不激活 App 的前提下粘贴到 Safari、Notes、Word、Finder 等目标。
- 固定窗口和非固定窗口都验证。
- 辅助功能被撤销后 UI 能提示并恢复。
- 多显示器/全屏 Space 定位正确。

### Phase 5：偏好、备份、更新

目标：恢复完整产品能力。

- AppKit 偏好窗口。
- 设置读写和即时生效。
- 旧版数据导入 UI。
- 备份导出/导入。
- 存储迁移和缓存清理。
- 更新策略。

退出条件：

- 设置全部可持久化并跨重启恢复。
- merge/overwrite 导入均有恢复路径。
- 更新链路有明确方案和签名验证。

### Phase 6：发布切换

目标：从 Tauri 版本平滑过渡。

- 保持 bundle id `com.ayangweb.eco-paste`。
- 首次启动迁移快照。
- 用户可见的数据升级提示。
- codesign、notarization、DMG。
- 旧版本回退路径。

退出条件：

- 旧库、旧备份、旧版 history 三类数据在干净 macOS 用户环境验证。
- 签名身份变化后权限和启动项仍可恢复。
- 更新失败不会破坏应用或数据。

## 3. 必须保留的行为契约

这些契约不能因 UI 技术变化而改变：

1. `content_hash = blake3("<kind>:<content>")`。
2. 文本 HTML/RTF 必须同时具备 plain 文本才入库。
3. 文件 `content` 为换行路径，`file_types` 为 `d/f` 串。
4. 图片 `content` 为 PNG 文件名，资源在 origin/thumbnails 分片目录。
5. 去重命中只 `use_count + 1` 并刷新 `updated_at`。
6. 收藏/置顶/备注/分组不刷新 `updated_at`。
7. 搜索长词 FTS5、短词 LIKE。
8. 置顶恒排前。
9. 收藏和置顶默认受清理保护。
10. 写回 guard 必须避免自身回环。
11. `.ecopastebak` 明文 ZIP 结构和加密容器字节布局不变。
12. 旧版导入只承诺历史记录，不扩大到未实现设置迁移。

## 4. 风险登记

### P0：阻断级

#### R1. nsgo 默认 UI 不是 AppKit 控件

影响：如果直接使用 nsgo `base` / `ui`，产品不满足“真正 AppKit 原生控件”要求。

缓解：

- 只用 `core.Native`。
- 每窗口一个 Native 根。
- 先做控件 spike。

#### R2. NSTableView delegate/data source 的 purego 实现风险

影响：列表是剪贴板产品核心，无法稳定实现则整个方案不成立。

缓解：

- 最先实现 1000 行列表 spike。
- 评估 cgo Objective-C bridge 作为兜底。
- 不在 spike 前确定 UI 框架。

#### R3. FTS5 trigram 驱动兼容性

影响：旧库搜索失效或索引损坏。

缓解：

- 选驱动前用真实旧库测试。
- 对比 FTS/LIKE 结果。
- 不重建旧 FTS 表。

#### R4. 加密备份字节不兼容

影响：用户无法恢复备份，属于数据灾难。

缓解：

- 保留 magic、header、参数、nonce 顺序。
- 用 Rust 样本做双向交叉测试。
- 任何格式调整都必须升级 format version，不能静默改变 v1。

### P1：高

#### R5. nsgo API 不稳定

影响：升级后编译失败或行为改变。

缓解：

- 锁定 pseudo-version/commit。
- 所有 nsgo 调用集中在适配层。
- 不在业务代码中散落 nsgo API。

#### R6. NSPanel key / 粘贴时序

影响：粘贴失败、按键被自家窗口吞掉、焦点跳到错误应用。

缓解：

- 真实应用矩阵测试。
- 固定/非固定分别验证。
- 用显式 window state 和事件而不是 sleep 猜测。

#### R7. Accessibility 权限和签名变化

影响：用户升级后粘贴不可用。

缓解：

- 保持 bundle id 和稳定签名身份。
- 首次启动/偏好页持续检测真实权限状态。
- 提供系统设置跳转和重试。

#### R8. 数据库 overwrite 导入的原子性

影响：替换失败导致当前数据库不可用。

当前 Rust 实现的实际行为是：暂停 watcher、关闭 pool、直接 `fs::copy` 备份数据库到 live 路径、删除 `-wal`/`-shm`、重新 `db::init`，然后再替换资源和设置。没有临时文件、原子 rename、旧库快照或失败回滚。

缓解：

- 迁移前快照。
- 临时目录 + 原子替换。
- 失败回滚旧 pool/manifest。
- 先暂停 watcher。

#### R9. 更新方案不兼容

影响：新版本无法自动更新或用户被留在旧版本。

缓解：

- 单独评估 Sparkle/Tauri/过渡更新器。
- 不把更新作为 UI 迁移的附属任务。
- 迁移期保留手动下载回退。

### P2：中

#### R10. 图片/SVG/富文本渲染差异

影响：视觉和交互回退。

缓解：

- 用 AppKit 原生 attributed string/image view。
- 对 HTML/RTF 做安全处理和资源限制。
- 用真实样本对照旧版。

#### R11. 输入法和键盘导航

影响：中文输入、搜索框、快捷键冲突。

缓解：

- 使用标准 `NSSearchField` / `NSTextField`。
- 不拦截输入法组合键。
- responder chain 和 menu validation 做专项测试。

#### R12. 多显示器和全屏 Space

影响：窗口出现在错误屏幕或不可见。

缓解：

- 复刻 follow/center/remember。
- 使用 `NSScreen.visibleFrame` 夹取。
- 测试 Retina、不同缩放、全屏辅助。

#### R13. 旧版 settings 不迁移

影响：用户升级后设置回到默认。

缓解：

- 明确产品承诺。
- 若后续要求迁移，单独设计旧 `.store.json` 映射和测试。

## 5. 需要用户后续决策的问题

这些不影响当前分析文档，但进入实现前必须明确：

1. 更新链路选择 Sparkle、Tauri 兼容协议，还是过渡双更新器。
2. 第一阶段是否保留 Windows 构建，还是只发布 macOS。
3. 旧版设置是否需要迁移，还是只迁移历史记录。
4. 主窗口是否继续“无 Dock 图标、菜单栏应用”模式。
5. 是否接受 cgo Objective-C bridge 作为 AppKit 复杂控件的兜底。
6. 是否要求保持当前 bundle id 和签名身份连续。
7. 历史数据首次打开时是否强制生成迁移前备份。
8. 是否保留 Tauri 版本作为并行回退版本。

## 6. 验证矩阵

### 数据

- 空库首次启动。
- 当前 Rust 库直接打开。
- 自定义数据目录。
- 旧版 `history` 导入。
- 明文备份 merge/overwrite。
- 加密备份 merge/overwrite。
- 图片缺失、缩略图缺失、文件路径失效。
- WAL 未 checkpoint 的库。

### 剪贴板

- 纯文本、长文本、HTML、RTF。
- URL、email、color、path。
- PNG、TIFF 截图、浏览器图片。
- 单文件、多文件、目录、失效文件。
- 密码/Token、排除应用。
- 自身复制/粘贴回环。
- 快速连续复制。
- 监听暂停。

### UI

- 1000+ 条列表滚动。
- 搜索短词、长词、CJK、大小写、混合空格。
- 收藏、置顶、备注、分组、删除、清空。
- 右键菜单和二级分组。
- 悬停/空格预览。
- 拖出到 Finder、浏览器、编辑器。
- 多显示器、全屏 Space、Retina。

### 系统

- 菜单栏图标。
- 全局快捷键冲突。
- 开机自启。
- Dock reopen。
- `.ecopastebak` 冷启动/运行中打开。
- Accessibility 权限授予/撤销。
- 更新下载、签名、安装失败。

## 7. 迁移完成定义

只有同时满足以下条件，才能称为“完成 Go + nsgo 迁移”：

1. 用户可见 UI 全部为 AppKit 原生控件，无 WebView。
2. 旧数据库、旧图片资源、旧备份可直接使用。
3. 旧版历史导入可用。
4. 主窗口不激活 App 也能粘贴。
5. 收藏、置顶、分组、备注、搜索、清理语义一致。
6. 备份 merge/overwrite 可恢复。
7. 权限、快捷键、托盘、拖出、更新在真实 macOS 上通过。
8. 有明确的回退和迁移前备份机制。

当前仅完成第 0 阶段的分析文档，不满足上述任何实现完成条件。
