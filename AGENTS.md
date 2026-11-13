<!-- magpie:begin · written by magpie from its Library -->
- 默认使用自然、简洁的语言与我交流，不限制具体语言。
- 只有在需要我回答问题、做出选择或进行确认时，才使用简体中文提出问题，并清晰列出选项；其他情况下，可根据任务需要自由选择合适的语言。
- 优先采用你推荐的方案，自主分析、决策并推进任务。
- 能独立解决的问题就自行解决，不要轻易向我提问；只有在确实需要我介入时，才向我寻求帮助。
- 遇到问题时，优先自行查阅资料、分析原因并尝试解决，不要轻易停下来或将问题交还给我。
- 表达简洁直接，避免不必要的解释和过程汇报，任务完成后汇报最终结果即可。
<!-- magpie:end -->

# AGENTS.md

> 本文件是 EcoPaste 所有 AI 编码工具的单一真相源。其它工具入口只应引用本文件，不要重复维护规则。
> 本文件也是仓库内唯一的 AI 指令文件。不要新增 `.trellis`、`.agents`、`.claude`、`.codex` 等工具目录。
> 迁移分析、阶段计划和功能对照表在 `docs/nsgo-migration/`。

## 0. 当前阶段

EcoPaste 正在从 **Tauri v2 + Rust + React** 迁移到 **Go + NSGo + AppKit**。

- **当前只支持 macOS。** 新架构不新增 Windows/Linux 代码、依赖、构建产物或兼容承诺。
- 仓库不再保留旧版可运行工程；只保留 `legacy/backend/` 剪贴板核心与 `legacy/frontend/` UI 参考。
- `legacy/` 不参与构建，不要求编译通过；不要为了让它编译而恢复 `src/`、`src-tauri/`、Cargo/Tauri 配置。
- 旧行为以 git history、迁移文档和 `legacy/` 参考代码为准。
- 当前工作以模块化实现、兼容性验证和持续交付为主；按垂直切片推进，不进行一次性大重写。
- 功能是否完成以 `docs/nsgo-migration/04-feature-parity-checklist.md` 为准，不以“代码看起来写完了”为准。
- 当前阶段验收标准：EcoPaste 能在 macOS 正常构建和运行，核心剪贴板功能完整，UI 与系统交互尽可能符合 macOS 原生体验。
- 旧项目只作为业务参考；功能取舍以产品判断为准，不要求逐项复刻。
- 架构边界允许未来扩展其他平台，但当前不为 Windows/Linux 预写平台适配代码；未来平台同样坚持原生 UI 优先。

### 0.1 产品设计与功能取舍

1. 不要求新版本完全复刻旧版本，旧项目仅作为业务参考，不是必须遵循的设计标准。
2. 根据自己的技术判断和产品理解，合理决定功能的保留、删除、简化或新增。
3. 优先考虑实际使用体验、功能实用性、操作效率和整体产品一致性。
4. 对于使用频率低、设计不合理、实现成本过高或没有必要保留的功能，可以主动精简。
5. 如果发现更合理的功能设计或交互方式，可以主动优化，不必受旧实现限制。
6. 核心目标是打造一款简洁、现代、易用且符合 macOS 原生体验的剪贴板管理工具，而不是机械地迁移旧代码。
7. 重要的功能调整需要同步更新功能清单和迁移文档，记录调整原因。

执行要求：

- 自主规划重构优先级，一次只做一个模块或一项相对独立的功能。
- 每完成一个模块，运行必要的格式化、构建、测试和静态检查。
- 确认没有明显问题后，及时提交并推送到迁移分支，然后继续下一个模块。
- 非关键决策自行调研并选择合理方案，不频繁停下等待确认。

## 1. Git 规范

### 1.1 分支

- `master` 是基线分支，禁止直接提交或推送。
- NSGo 迁移全程使用长期分支 `refactor/nsgo-migration`，从最新 `master` 创建一次；不按 Phase 另开分支。
- 迁移分支只承载 NSGo 迁移相关改动；无关的独立功能/修复仍走 `feat/<scope>`、`fix/<scope>`。
- 不把无关格式化、依赖升级或历史清理混入迁移提交。
- 迁移模块完成并验证后，按用户授权提交并推送到 `refactor/nsgo-migration`；不推送 `master`、不创建 PR、不打 tag、不触发 release。
- 迁移分支合并/切回 `master` 只在 parity 通过后执行。

### 1.2 提交信息

- 遵循 Conventional Commits，使用单行格式：

```text
<type>(<scope>): <summary>
```

- 常用 type：`feat`、`fix`、`refactor`、`docs`、`test`、`chore`、`build`、`perf`。
- summary 使用祈使语气，准确描述结果，不写“update code”“fix bug”“wip”等无意义信息。
- 一个独立功能、一个迁移阶段或一个可验证修复一个提交；不要积累大量无关改动后一次性提交。
- 提交前必须通过该阶段要求的检查；不要把明显不可构建的状态提交到共享分支。
- 不在提交信息中写内部阶段编号、临时 TODO 或未验证的承诺。

示例：

```text
feat(clipboard): add native pasteboard watcher
fix(history): preserve updated_at on metadata updates
docs(migration): record NSGo parity checks
refactor(settings): isolate JSON patch merging
```

## 2. 代码规范

### 2.1 基本要求

- 保持代码简洁，函数和类型职责单一。
- 优先复用已有实现；重复逻辑先提取，不复制粘贴。
- 优先早返回，减少嵌套；主流程保持线性可读。
- 只写解释“为什么”的注释，不写复述代码的注释；不保留历史残留注释、阶段号或外部行号。
- 命名使用完整英文单词；Go 导出标识符按 Go 规范，包名短小且语义明确。
- 不做超出当前需求的抽象、兼容垫片或提前优化。

### 2.2 文件与模块边界

- 单个 Go 文件以 300–500 行为参考上限；超过 600 行必须拆分，生成代码、纯数据表或迁移定义除外。
- 按职责拆分文件，不按“少建文件”拆分；不要为了减少文件数量把仓储、业务、桥接混在一个文件。
- 目录边界：

```text
docs/nsgo-migration/       # 迁移分析、计划与验收文档
legacy/                    # 旧剪贴板核心与前端 UI 参考，不参与构建
cmd/ecopaste/              # 入口与 wiring
internal/app/              # 应用编排、窗口生命周期、服务组装
internal/ui/               # NSGo 视图：布局、状态、事件转发
internal/native/macos/     # AppKit/CoreGraphics 桥接，叶子层
internal/clipboard/        # 采集、写回、去重、敏感内容
internal/database/         # SQLite、migration、仓储
internal/settings/         # 设置模型、patch、持久化
internal/backup/           # .ecopastebak
internal/storage/          # 数据目录、图片/图标存储
internal/keystroke/        # 粘贴注入
internal/menu/             # 原生菜单构建
internal/tray/             # 托盘
internal/shortcut/         # 全局快捷键
internal/update/           # Sparkle/appcast
internal/i18n/             # 文案目录
internal/onboarding/       # 权限与旧版导入
```

- 依赖方向必须单向：`cmd` → `app` → 业务模块；`native` 是叶子层，不读数据库、不写设置、不依赖 UI。
- `ui` 不直接访问 SQLite、文件系统或 AppKit 原始指针；通过 app service 接口。
- 业务模块不依赖 AppKit；平台能力通过窄接口注入。
- 接口定义在消费方，避免为未来平台预建大而全的抽象。

### 2.3 Go 约定

- Go 版本与依赖以 `go.mod` 为准；不要引入需要 CGO 的依赖，NSGo macOS 后端要求 `CGO_ENABLED=0`。
- 错误使用 `fmt.Errorf("...: %w", err)` 包装；用户可读错误与技术上下文分开。
- 不用 `panic` 处理可预期错误；初始化失败返回错误并在入口处理。
- IO、数据库、网络、外部命令必须接收 `context.Context` 并尊重取消。
- 并发任务必须有明确 owner 和退出路径；不要启动无法停止的 goroutine。
- 共享状态优先通过 channel 或明确锁保护；不要用“应该不会同时发生”作为并发假设。
- 数据库写入使用单写 worker 或受控连接；不要在 AppKit 主线程执行长事务。
- 类型优先使用明确 struct、枚举常量和 `time.Time`；避免无约束 `any` 和字符串魔法值。
- 跨模块复用的字面量集中为常量，不散落重复字符串。

### 2.4 旧栈代码

- `legacy/` 只作为行为参考，不是构建目标；优先读代码提取契约，不直接复制 Rust 实现。
- `legacy/frontend/` 只用于对照页面、组件、文案、样式和交互；不要恢复 WebView/Tauri 构建。
- 不在 `legacy/` 上新增功能或修复旧 bug；迁移中的新实现写 Go/NSGo。
- 需要查旧行为时，先读 `legacy/README.md` 的保留范围和数据契约文档。

## 3. macOS UI 规范

### 3.1 原生优先

- **优先使用 AppKit 原生控件和系统能力。**
- 页面布局、文案、状态和交互顺序可参考 `legacy/frontend/`，但实现必须用 AppKit。
- NSGo 负责应用结构、状态、布局和事件编排；不要用 NSGo 通用 UI 组件替代 AppKit 控件。
- 不使用 WebView、React、Ant Design、UnoCSS 或 CSS 实现新 UI。
- 能用系统控件表达时，不自定义绘制；自定义视图只用于系统控件无法满足的卡片/预览等场景。

### 3.2 推荐映射

| 需求 | 首选 |
| --- | --- |
| 搜索 | `NSSearchField` |
| 分类/分段 | `NSSegmentedControl` |
| 下拉/排序 | `NSPopUpButton` |
| 开关 | `NSSwitch` |
| 按钮 | `NSButton` |
| 文本输入 | `NSTextField` / `NSTextView` |
| 历史列表 | `NSTableView` / `NSCollectionView` |
| 图片 | `NSImageView` / `NSImage` |
| 进度 | `NSProgressIndicator` |
| 弹窗 | `NSAlert` / `NSPanel` |
| 文件选择 | `NSOpenPanel` / `NSSavePanel` |
| 菜单 | `NSMenu` |
| 托盘 | `NSStatusItem` |
| 拖出 | `NSDraggingSession` + `NSPasteboardItem` |
| 材质/背景 | `NSVisualEffectView` |
| 颜色 | `NSColor.labelColor` 等语义色 |

### 3.3 原生体验

- 使用系统语义色、系统字体、系统间距和 `NSAppearance`，不要硬编码浅色/深色。
- 窗口、菜单、弹窗、工具栏、文件选择器、快捷键录入优先使用系统 API。
- 剪贴板面板使用 `NSPanel`，保持非激活、可成为 key、失焦隐藏等原生行为。
- 尊重系统减少动态效果、键盘导航、VoiceOver 和全键盘操作。
- 自定义视图必须提供 `accessibilityLabel`、`accessibilityRole`、`accessibilityHelp`。
- 改 UI 后必须实际操作主路径和边界，不能只靠编译通过。

### 3.4 NSGo 能力不足时

- 在 `internal/native/macos/` 增加窄桥接，不在 `internal/ui/` 里堆平台细节。
- 桥接层只暴露业务需要的最小接口；不要让业务代码直接依赖 `core.Native` 的 props/命令细节。
- 不为“可能以后需要”的跨平台差异预建抽象；当前只实现 macOS。
- 如果 NSGo 组件行为与 AppKit 冲突，优先保留 AppKit 原生行为。

## 4. 第三方库规范

- UI 层优先使用 AppKit；不要为了统一外观引入跨平台 UI 库。
- 非 UI 能力只要存在成熟、合适的开源库，就优先使用，不重复造轮子。
- 引入依赖前评估：
  - 维护活跃度与发布节奏；
  - License 是否允许分发；
  - macOS/arm64 支持；
  - 是否需要 CGO；
  - 安全记录与传递依赖；
  - 能否被薄封装替换。
- 依赖版本必须可复现：Go module 固定版本；NSGo 固定 commit；不使用 floating branch 或 `latest`。
- NSGo 升级单独提交，并跑完整兼容矩阵；升级说明写入迁移文档。
- 自研只用于：没有合适库、库无法满足原生体验、或桥接成本明显更低。自研前先查 NSGo 官方文档和源码。
- 新增非显然依赖时记录原因、替代方案和移除条件。

## 5. 数据与安全

### 5.1 已发布契约

以下内容按“已发布数据”处理，不能随意改：

- SQLite schema、索引、FTS5 trigram 表和触发器。
- `settings.json` 的 camelCase 字段、默认值和 patch/merge 语义。
- 数据目录：`<app_local_data>/<dev|prod>`、`storage.json` 锚点、自定义目录。
- `content_hash`：`blake3("<kind>:<content>")`。
- 图片文件名与 `resources/clipboard-images/{origin,thumbnails}` 分片路径。
- `updated_at`：表示内容最近一次被复用；备注、收藏、置顶、分组更新不得刷新。
- `.ecopastebak` 明文 ZIP 与加密容器格式。
- Bundle ID、文件关联扩展名与 UTI。

### 5.2 变更规则

- 已发布 migration 不回改；schema 变更必须新增 migration。
- 改 schema 时同步检查所有 `SELECT`、`INSERT`、`UPDATE`、`bind` 和测试结构体。
- 数据变更必须有迁移、回滚和备份策略；不确定时先只读检测，不直接写。
- 设置写入使用临时文件 + rename 原子替换；数据库写入使用事务。
- 数据契约变更必须保证旧数据与备份仍可读；旧工程已归档，不要求同仓库双栈构建。

### 5.3 安全

- 禁止在日志、崩溃报告、测试快照中记录完整剪贴板内容、密码、token、私钥。
- 敏感内容识别与脱敏逻辑必须可测试；默认不采集私钥/密码类内容。
- 备份密码、Sparkle 私钥、签名证书不得进入仓库或 CI 日志。
- 开发版必须使用独立 bundle ID 和独立数据目录，禁止读写正式用户数据。
- 数据兼容测试只使用真实数据的副本，不直接操作生产目录。

## 6. 测试与验收

### 6.1 自动检查

Go/NSGo 变更至少运行：

```bash
gofmt -l .
go vet ./...
go test ./...
```

涉及并发、剪贴板监听、数据库 worker 时追加：

```bash
go test -race ./...
```

涉及 NSGo 视图、窗口或打包时追加：

```bash
nsgo check
nsgo test
nsgo build
```

`legacy/` 不参与自动检查；不要为了跑旧检查而恢复旧前端或 Tauri 工程。

### 6.2 手工验收

自动检查不能替代 macOS 真机验收。至少覆盖：

- 冷启动、热启动、退出、重启。
- 全局快捷键唤起/隐藏、托盘、菜单。
- 文本/HTML/RTF/图片/文件复制与粘贴。
- 重复复制去重、点击粘贴无回环。
- 搜索长词/短词、收藏/置顶/备注/分组、清理。
- 预览、拖出、备注编辑、窗口固定/失焦隐藏。
- 权限允许/拒绝、自启、数据目录切换、备份导入导出。
- 深浅色、中文/英文、VoiceOver 基本可用。

### 6.3 完成定义

- 不把“编译通过”当作功能完成。
- 不把“旧版看起来一样”当作数据兼容证明。
- 每个迁移项在 parity 清单中更新状态，并记录验证方式。
- 新增差异必须写明原因，不允许静默改变旧行为。

## 7. 迁移执行规则

- 按 `docs/nsgo-migration/03-migration-plan.md` 的阶段推进，一次只做一个垂直切片。
- 先数据契约，再剪贴板管线，再 UI，再发布链路。
- UI 迁移先对照 `legacy/frontend/src/pages` 与 `components`，再按 parity 清单逐项验收。
- 旧工程已移出仓库；新栈使用独立 bundle id 与数据目录。
- `legacy/` 只在参考价值消失后删除；删除前确认迁移文档不再依赖对应实现。
- 不为了迁移而盲目推翻已有业务逻辑；能复用规则、文案、数据结构的先复用。
- 不新增 Windows/Linux 抽象；未来平台适配单独立项。
- 迁移中发现的隐藏耦合、行为差异和临时方案，记录到 `05-risk-register.md` 或对应任务。

## 8. 文档与决策

- 迁移相关事实写在 `docs/nsgo-migration/`，不要只留在聊天记录里。
- 功能完成情况更新 `04-feature-parity-checklist.md`。
- 新的架构取舍、依赖选择、桥接方案记录为 ADR 或迁移文档小节。
- 文档中的路径、命令、行为必须基于真实代码；不要凭空设计。
- 外部参考：
  - NSGo：<https://github.com/nsgo-dev/nsgo>
  - AppKit：<https://developer.apple.com/documentation/appkit>
  - Human Interface Guidelines：<https://developer.apple.com/design/human-interface-guidelines/>
  - Go：<https://go.dev/doc/>

## 9. 通用协作

- 尊重 dirty worktree；不要回滚或覆盖非本轮改动。
- 需要动到已修改文件时先读清楚，再做最小改动。
- 发现与当前任务无关的 bug，不顺手修；在最终汇报中说明。
- 不提交构建产物、用户数据、日志、备份包或本地临时文件。
- 仓库内只保留 `AGENTS.md` 这一份 AI 指令文件，不恢复已移除的工具目录。
- 任务完成后汇报：做了什么、验证了什么、哪些未验证、下一步是什么。
