# 03 · 渐进式迁移计划

> 目标：在不丢失现有功能与数据的前提下，把 EcoPaste 从 Tauri v2 + Rust + React 迁移到 Go + NSGo + AppKit。
> 当前范围：macOS。Windows/Linux 不进入新架构的实现与验收。
> 旧工程已移出仓库；下文旧代码路径均指 `legacy/backend/...`、`legacy/frontend/...` 保留部分或 git history。

## 1. 执行原则

1. **先兼容，再替换。** 数据目录、SQLite schema、`settings.json`、`content_hash`、图片文件名和 `.ecopastebak` 必须先兼容，再讨论 UI。
2. **单运行时目标。** 长期架构只保留 Go 进程；不采用 Rust sidecar、Rust staticlib + cgo、或 Tauri 内嵌 NSGo。
3. **归档可回退。** 旧完整工程已移出仓库；迁移期以 `legacy/` 参考代码、迁移文档和 git history 作为回退依据，不在仓库维护双栈构建。
4. **UI 原生优先。** NSGo 负责窗口结构、状态、布局和事件编排；可见控件与系统交互优先使用 AppKit。
5. **一次只切一条主路径。** 每个阶段必须能独立运行和验证，不允许“全部写完再联调”。
6. **不提前抽象。** 只为当前 macOS 需求定义接口；未来平台适配在真正需要时再引入。

## 2. 分支与提交策略

| 项目 | 规则 |
| --- | --- |
| 基线分支 | `master` |
| 工作分支 | `refactor/nsgo-migration`，从最新 `master` 创建一次的长期集成分支 |
| 分支策略 | 所有 NSGo 迁移阶段都在此分支进行，不按 Phase 另开分支 |
| 提交粒度 | 一个阶段、一个垂直切片或一个可独立验证的修复一个提交 |
| 提交格式 | Conventional Commits 单行，如 `feat(clipboard): add native pasteboard watcher` |
| 禁止 | 直接提交 `master`、混合无关改动、提交构建产物、提交用户真实剪贴板数据 |
| 检查点 | 每个 Phase 结束时打本地 tag 或记录 commit SHA，便于回滚与对比 |
| 发版 | 旧 Tauri 流水线已归档；NSGo 从 Phase 6 建立新构建，当前不发版 |
| 合并 | parity 通过并获批后才把迁移分支合并回 `master` |

建议阶段提交：

```text
docs: add NSGo migration analysis
chore: remove ignored build artifacts
feat(app): bootstrap Go and NSGo shell
feat(clipboard): add macOS pasteboard and watcher
feat(history): port SQLite repositories and search
feat(ui): add native history list
feat(settings): port settings and preferences UI
feat(release): add Sparkle appcast packaging
```

## 3. Phase 0 · 基线与兼容夹具

**目标**：在任何 Go 代码落地前，建立可重复验证的“旧版真实行为”基线。

### 工作项

1. 固定 NSGo：
   - 模块：`github.com/nsgo-dev/nsgo`
   - 基线 commit：`70cef6a4ed17f7c8428faabc9bb419ff4223745b`（2026-11-12）
   - Go 要求：`go 1.26.0`；本机验证版本 `go1.27.4 darwin/arm64`
   - `go.mod` 使用固定 commit；升级必须单独提交并跑完整兼容矩阵。
2. 建立数据夹具（不入库）：
   - 复制真实 `prod/` 数据目录到仓库外测试目录。
   - 至少准备：空库、1k 条、10k 条、含 FTS 特殊字符、含 1–2 字符关键词、含收藏/置顶/分组/备注、含图片/文件/HTML/RTF/敏感内容。
   - 记录每份夹具的 schema、行数、查询结果和文件清单。
3. 录制旧版行为：
   - `list_clipboard_items` 在固定筛选条件下的 JSON 输出。
   - `search` 在长/短关键词下的结果顺序。
   - `content_hash`、图片文件名、缩略图尺寸、来源应用图标映射。
   - `.ecopastebak` 明文与加密样本；加密样本只记录可解析结构和 round-trip 结果，不记录密码。
4. 建立兼容测试骨架：
   - Go 侧读取旧 DB 副本的只读测试。
   - 备份 round-trip 测试。
   - 设置 JSON patch/merge 测试。
5. 记录旧版构建基线（仅归档保留，当前仓库不执行）：
   - 旧工程构建结果见归档与 git history；当前仓库只保留参考代码。
   - 不为了跑旧检查恢复可运行的 `package.json`、`Cargo.toml` 或 `src-tauri/`。

### 退出条件

- 兼容夹具可一键复制和清理。
- 任意一份旧 DB 都能被测试代码打开并完成只读查询。
- 已记录“旧版正确输出”，后续不需要靠记忆判断 parity。

## 4. Phase 1 · Go + NSGo 应用壳

**目标**：建立可启动、可退出、可唤起的最小 macOS 应用，不写入用户数据。

### 工作项

1. 建立工程：

```text
go.mod
nsgo.yaml
cmd/ecopaste/main.go
internal/app/
internal/ui/
internal/native/macos/
```

2. 开发期隔离：
   - 独立 bundle id，例如 `com.ayangweb.eco-paste.dev`。
   - 独立数据目录，禁止读写现有 `prod/`。
   - 不上传、不覆盖旧版设置。
3. 应用壳：
   - `nsgo.Run` 启动。
   - 无边框 `NSPanel` 作为剪贴板主面板。
   - 托盘图标与最小菜单。
   - `Alt+C` 全局快捷键唤起/隐藏。
   - `NSSearchField` + 关闭按钮的最小布局。
   - `desktop.OnReopen` 与退出清理。
4. 工程边界：
   - `internal/native/macos` 不依赖数据库和设置。
   - `internal/ui` 不直接访问 SQLite/文件系统。
   - `cmd/ecopaste` 只做 wiring。

### 退出条件

- `go build ./...`、`go vet ./...`、`gofmt -l .` 通过。
- `nsgo check` 通过。
- 应用可启动、显示面板、托盘可退出。
- `Alt+C` 可稳定唤起，不抢错焦点。
- 不产生任何业务数据写入。

## 5. Phase 2 · 剪贴板垂直切片

**目标**：打通“监听 → 读取 → 识别 → 入库 → 列表 → 写回 → 粘贴”的最小闭环。

### 工作项

1. `internal/native/macos/pasteboard`：
   - `NSPasteboard.changeCount` 轮询，间隔 120ms。
   - 读取 plain text、HTML、RTF、PNG/TIFF、file URL。
   - 多表示同时存在时按 `capture.order` 选择主类型，同时保留可写回表示。
   - 写入时使用单个 `NSPasteboardItem`，一次写入 plain + HTML/RTF 或多文件 URL。
2. `internal/clipboard`：
   - 采集优先级、类型开关、大小上限。
   - `blake3("<kind>:<content>")` 去重。
   - 图片 PNG 直通/重编码、原图与缩略图落盘。
   - 文件路径换行拼接与 `file_types`。
   - 来源应用同步抓取；首次遇到应用时取原生图标，降采样为 128px PNG 放入 `resources/app-icons/`，按内容哈希去重，启动时清理无引用文件。
   - 敏感内容识别与忽略应用。
   - 回环抑制：写回前登记所有实际写出的表示 hash（记录内容、plain fallback、HTML、RTF），2 秒 TTL。
3. `internal/database`：
   - `modernc.org/sqlite`，WAL、foreign_keys、busy timeout。
   - 识别现有 `_sqlx_migrations`，把 `0001` 标记为已应用，不重复建表。
   - 首次运行先备份 DB，再建立 Go 侧 migration 记录。
4. `internal/keystroke`：
   - `CGEvent` 发送 ⌘V。
   - 粘贴前 hide 或 resign key；固定窗口粘贴后恢复 key。
   - 辅助功能权限检查与引导。
5. 最小列表：
   - 只显示最近 N 条，验证入库、排序、缩略图和来源图标。

### 必须通过的场景

| 场景 | 预期 |
| --- | --- |
| 复制纯文本 | 新增一条 `text` 记录，`search_text` 正确 |
| 复制网页富文本 | 主类型按设置选择，HTML/RTF 可写回 |
| 复制图片 | 原图与缩略图落盘，列表显示缩略图 |
| 复制多个文件 | `content` 为换行路径，`file_types` 正确 |
| 重复复制同一内容 | 不新增记录，`use_count + 1`，`updated_at` 刷新 |
| 点击历史项粘贴 | 目标 App 收到内容，不新增回环记录 |
| 暂停监听 | 复制不入库，恢复后继续 |
| 忽略应用 | 该应用复制不入库 |
| 敏感内容 | 按设置跳过或脱敏，日志不出现原文 |

### 退出条件

- 旧 DB 副本可直接打开并新增记录。
- 新 DB 可被旧版 Tauri 应用打开（只读验证至少通过）。
- 剪贴板格式矩阵全部通过。
- 连续复制 100 次无重复、无丢失、无回环。

## 6. Phase 3 · 历史、搜索与整理

**目标**：达到旧版历史管理能力。

### 工作项

1. 查询与分页：
   - kind/group/favorite/pinned/group filter/keyword/sort/limit/offset。
   - `list + total + has_more` 一次返回。
   - 列表不返回完整 HTML/RTF content，只返回 summary。
2. 搜索：
   - FTS5 trigram，≥3 字符走 FTS。
   - 1–2 字符走 `LIKE`，中文按字符计数。
   - 关键词转义，避免 FTS 语法错误。
   - `defaultFocus` 决定面板打开时聚焦搜索框还是历史列表；`clearOnHide` 控制隐藏后是否清空关键词并恢复完整列表。
3. 整理：
   - 收藏、置顶、备注、分组、分组布局、移动到分组。
   - 备注/收藏/置顶/分组更新不刷新 `updated_at`。
   - 删除条目时清理图片文件。
4. 清理与统计：
   - 保留时长 + 最大条数；收藏/置顶保留。
   - 启动清理、定时清理、清空记录。
   - 存储占用、缓存清理。
5. 来源应用与图标：
   - 应用注册、未引用清理、从路径添加。
   - `NSWorkspace` 图标抽取与缓存。
6. 旧版数据：
   - 产品调整：macOS 首版不提供 v0.6 数据目录导入；旧版用户先导出 `.ecopastebak`，再通过新版备份导入恢复。

### 退出条件

- 固定夹具上的查询结果与旧版逐项一致。
- 10k 条数据下列表首屏与翻页无可感知卡顿。
- 收藏/置顶/备注/分组操作不改变复用时间。
- 清理后 DB 引用与磁盘文件一致，无孤儿图片。

## 7. Phase 4 · 原生历史 UI 与预览

**目标**：用 AppKit 实现主交互，不依赖 WebView。

### 工作项

1. `NSTableView` / `NSCollectionView`：
   - 原生虚拟化、选中态、键盘上下移动、PageUp/PageDown、Home/End；同时兼容 macOS 的 Fn+↑/↓ 翻页。
   - 文本/图片/文件卡片使用 `NSTableCellView` 或自定义 `NSView`。
   - 空状态、加载态、分组切换。
2. 搜索与筛选：
   - `NSSearchField` 实时搜索，防抖。
   - `NSSegmentedControl` 分类，`NSPopUpButton` 排序/分组。
3. 预览：
   - 原生 `NSPopover`（产品调整）：锚定所选行弹出，保持面板可操作；临时 popover 支持点击外部、Space、Esc 关闭，避免独立窗口抢焦点和多屏定位复杂度。
   - `NSImageView` 图片预览；文本/文件使用只读 `NSTextView`，支持选择复制。
   - Space 打开或关闭、Esc 返回；预览态下回车沿用历史复用路径，`⌘C`/`⌥⌘C` 等快捷键继续作用于所选行；敏感文本按设置脱敏。hover 预览暂不实现，按实际反馈再评估。
4. 右键菜单：
   - `NSMenu` 动作分组、快捷键、分组子菜单。
   - 菜单动作回到 Go 业务层，不在 UI 内写业务逻辑。
   - 按内容动态精简菜单：URL/邮件提供打开，路径/文件提供 Finder 定位，图片提供另存，避免所有条目堆同一套无关动作。
   - 文本提供“纯文本粘贴/复制为纯文本”，文件提供“粘贴为路径”；动作只在对应内容类型出现，避免菜单噪音。
   - 标准编辑菜单的复制/粘贴作用于当前历史行；`⌘C`、`⌘V`、`⌥⌘C`、`⌥⌘V` 与右键菜单动作保持一致。
   - 备注使用原生模态编辑，移除备注会归一为 NULL；非空备注遵循 `autoFavorite` 联动收藏。
5. 拖出：
   - `NSDraggingSession` + `NSPasteboardItem`。
   - 文件、图片、纯文本、HTML/RTF 多格式。
6. 交互细节：
   - 单击/双击/中键动作。
   - 悬停快捷动作及顺序。
   - 备注内联编辑。
   - 固定窗口、失焦隐藏、编辑中不隐藏。
   - NSGo 的 `HideOnBlur` 无法临时暂停，改为在原生面板监听 resign-key 后回传隐藏事件；系统保存面板等 modal 操作期间暂停隐藏。
   - 图片另存使用 app-modal `NSSavePanel`，并提升到剪贴板浮动面板层级之上；不把文件面板嵌进置顶面板，避免被遮挡或连带隐藏。

### 分组产品调整

- 首版保留分组的创建、重命名、删除、条目移动与筛选，不迁移 SVG 图标编辑器、隐藏分组和拖拽排序。
- 自定义分组按创建顺序稳定展示；新分组追加到末尾。图标与 `is_hidden`、`sort_order` 字段继续按已发布 schema 读写，避免破坏旧数据。
- 分组管理收进面板内的原生操作菜单，减少独立管理窗口和复杂布局；删除分组时条目保留并回到“未分组”。
- 该取舍优先保证主路径效率与 macOS 原生交互一致性；若真实使用中出现排序需求，再基于反馈增加简单上移/下移，而不恢复旧版复杂管理页。

### 提示音产品调整

- 旧版把 24KB `copy.mp3` 打入二进制并依赖 `rodio` 解码；新版使用系统 `NSSound` 的 `Pop` 音效，避免引入音频解码依赖和额外资源文件。
- 提示音只在监听或手动读取真正入库成功后播放，跟随 `clipboard.feedback.copySound`；播放失败不影响入库与列表刷新。

### 退出条件

- 主路径全键盘可完成：唤起 → 搜索 → 选择 → 粘贴。
- 1k/10k 条滚动稳定，无整表重绘。
- 预览不抢焦点、不闪烁、不越界。
- 菜单、拖出、备注、分组在真实 App 中可用。

## 8. Phase 5 · 设置与系统集成

**目标**：覆盖日常系统行为。

### 工作项

1. 设置：
   - Go struct 与 `settings.json` camelCase 完全兼容。
   - 深度合并 patch，数组整体替换，跨字段校验。
   - `.tmp` + rename 原子写。
   - 7 个偏好 Tab 的设置项、搜索、高亮定位。
2. 引导：
   - 产品调整：macOS 首版不提供首启引导浮层、旧版导入步骤和 `internal/onboarding` 包，冷启动直接进入日常使用。
   - 开发阶段不承担旧版兼容，`settings.json` 的 `onboarding` 字段与相关结构一并移除。
   - 权限缺失在粘贴动作与设置“权限”页提示，快捷键/排除应用/备份恢复分别由设置页和备份导入承担。
3. 权限：
   - 辅助功能、完全磁盘访问、通知（如使用）。
   - 权限状态检测、系统设置跳转、失败提示。
4. 系统集成：
   - 托盘菜单、应用菜单、Dock 显隐。
   - 全局快捷键录制、暂停/恢复、冲突提示。
   - 开机自启 `SMAppService`。
   - 单实例、`OnOpenFiles`、`.ecopastebak` 文件关联。
   - 数据目录切换、恢复默认与热迁移；恢复默认前保留旧默认目录的带时间戳备份。

### 退出条件

- 旧 `settings.json` 可被读取、修改、写回，字段不丢失。
- 首次启动全流程可完成，权限拒绝时有可恢复路径。
- 托盘、快捷键、自启、文件关联、数据目录切换全部通过人工验收。

## 9. Phase 6 · 备份、更新、发布与诊断

**目标**：补齐数据安全与发布链路。

### 工作项

1. 备份：
   - 明文 ZIP 与加密容器字节级兼容。
   - `merge` / `overwrite` 导入策略。
   - 打开文件、拖放接收、pending slot。
   - 大文件流式处理，避免整包进内存。
2. 更新：
   - Sparkle appcast + EdDSA 签名。
   - stable/beta/nightly 渠道映射。
   - 下载进度、安装、重启、跳过版本。
   - 保留旧 Tauri update endpoint 到 cutover。

   实现调整：macOS 首版只用单一 stable appcast，不恢复独立更新窗口；设置、应用菜单和托盘共用检查入口，下载/安装/重启/跳过由 Sparkle 管理。beta/nightly 需要独立 appcast 与签名渠道，单独立项后再接入。
3. 其他：
   - 复制音效、i18n（zh-CN/en-US）、日志轮转、诊断信息。
   - 缓存清理、存储占用、图片另存、Finder reveal、打开链接/邮件。
4. 发布：
   - `nsgo build` 生成 `.app` / `.dmg`。
   - 应用图标由 `cmd/ecopaste/assets/Icon.png` 生成 `AppIcon.icns`，同步用于 Dock、设置侧栏、关于窗口与 DMG。
   - Developer ID 签名、公证、Sparkle 签名。
   - GitHub Actions 增加 macOS Go/NSGo 流水线，不破坏旧流水线。

### 退出条件

- 旧版导出的明文/加密备份可在 NSGo 版导入，反之亦然。
- Sparkle 可从测试 appcast 完成一次真实升级。
- 签名、公证、安装、首次启动通过。

## 10. Phase 7 · Parity 验收与旧工程封存

### 工作项

1. 按 `04-feature-parity-checklist.md` 逐项验收。
2. 真实数据副本上做至少一轮完整迁移演练：
   - 旧版启动 → 产生数据 → 导出备份 → NSGo 版打开 → 查询/写入/再导出。
3. 内部 dogfood：
   - 至少一周日常使用，覆盖复制、搜索、粘贴、分组、清理、备份、更新。
4. 发布 beta：
   - 独立下载入口，不替换 stable。
5. 退役条件满足后：
   - 确认旧工程归档完整，仓库内不再恢复 Tauri 构建。
   - 按条件删除 `legacy/`。
   - 更新 README、release 文档和 AGENTS.md。

### 切换门槛

- `04` 中所有 P0/P1 项通过；P2 项有明确例外记录。
- 无未解决的数据丢失、回环、粘贴失败、崩溃问题。
- 备份双向兼容通过。
- 更新链路可用。
- 至少一个 beta 周期无阻断性回归。

## 11. 测试矩阵

| 层级 | 内容 | 工具/方式 |
| --- | --- | --- |
| Go 单元 | 类型识别、敏感内容、hash、设置 merge、备份 header、路径 | `go test ./...` |
| 数据集成 | 旧 DB 副本、FTS/LIKE、分页、触发器、清理 | 固定夹具 + SQL 断言 |
| 剪贴板矩阵 | text/html/rtf/image/files、多格式并存、空内容、超大内容 | 手工 + 可自动化的 pasteboard 测试 |
| 系统集成 | 快捷键、托盘、菜单、权限、自启、文件关联、单实例 | macOS 手工验收 |
| UI | 键盘导航、滚动、预览、拖出、备注、分组、主题 | `nsgotest` + 手工验收 |
| 备份 | 明文/加密 round-trip、旧版互操作、损坏包 | Go 测试 + 样本 |
| 更新 | appcast 解析、下载、签名校验、安装 | Sparkle 测试源 |
| 性能 | 启动、10k 列表、搜索、图片预览、内存 | Instruments + 基准测试 |
| 可访问性 | VoiceOver 标签/角色、键盘焦点 | Accessibility Inspector |
| 发布 | 签名、公证、安装、升级 | 测试机 |

### 每次提交的最低门槛

```bash
gofmt -l .
go vet ./...
go test ./...
```

涉及并发时追加：

```bash
go test -race ./...
```

涉及 NSGo 视图或打包时追加：

```bash
nsgo check
nsgo test
nsgo build
```

`legacy/` 不参与自动检查；不要为了跑旧前端或 Rust 检查恢复旧工程。

## 12. 停止条件与回滚

出现以下任一情况，停止当前阶段，不继续叠加改动：

- 旧 DB 副本出现写入失败、schema 漂移或 FTS 结果异常。
- 同一内容重复入库或粘贴产生回环记录。
- 备份无法 round-trip，或旧版无法读取新导出包。
- AppKit 主线程崩溃、窗口无法恢复、权限流程卡死。
- 真实数据目录被开发版读写。

回滚方式：

1. 保留旧 Tauri 应用与数据目录不动。
2. 删除开发版独立数据目录，重新从夹具开始。
3. 回退到上一阶段 commit/tag。
4. 记录失败场景到 `05-risk-register.md` 和对应任务，不靠口头记忆。

## 13. 阶段完成定义

一个阶段只有同时满足以下条件才算完成：

- 代码通过该阶段要求的自动检查。
- 对应 parity 项已更新状态。
- 主路径和边界在 macOS 真机手工验证。
- 数据契约未被破坏，或已有明确 migration。
- 文档记录了新增差距、临时方案和回滚点。
- 变更已按阶段提交，提交信息准确。
