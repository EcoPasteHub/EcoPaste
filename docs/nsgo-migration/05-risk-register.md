# 05 · 迁移风险登记

> 评级：影响 / 概率使用 高、中、低。`触发条件` 是必须停下来复盘的信号，不是普通待办。
> 本文只登记与本次迁移直接相关的风险；产品功能风险在 parity 清单中跟踪。

## 1. 最高优先级风险

| 排名 | 风险 | 为什么先处理 |
| --- | --- | --- |
| 1 | 数据契约被破坏 | 会直接丢用户历史，且不可逆 |
| 2 | NSGo API 漂移 | 可能导致整层 UI 返工 |
| 3 | AppKit 主线程/内存错误 | 表现为随机崩溃，最难排查 |
| 4 | 剪贴板回环/粘贴时序 | 直接破坏主路径可信度 |
| 5 | 备份与更新链路不兼容 | 用户无法恢复或升级 |

## 2. 风险登记表

| ID | 类别 | 风险 | 影响 | 概率 | 缓解措施 | 触发条件 |
| --- | --- | --- | --- | --- | --- | --- |
| R-01 | 框架 | NSGo 0.x API 仍可能变动（已发布 v0.1.0/v0.2.0 稳定版本） | 高 | 中 | 固定稳定 tag；升级先评估官方新 API/特性；`internal/native` 与 `internal/ui` 做适配层；升级单独提交并跑完整矩阵 | NSGo 升级导致编译失败或行为变化 |
| R-02 | 框架 | NSGo 内部实现变更影响 `core.Native`、`desktop`、`Scope` | 高 | 高 | 业务层不直接依赖 NSGo 细节；adapter 收口；保留替换能力 | 需要改业务代码才能适配 NSGo 升级 |
| R-03 | 工具链 | NSGo 要求 Go 1.27.3+，本机/CI 版本不一致 | 中 | 中 | `go.mod` 固定最低版本；CI 固定 Go 版本；记录验证版本 | 本机能构建、CI 失败或反之 |
| R-04 | 原生桥 | `purego` + Objective-C runtime 调用复杂，签名/内存管理易错 | 高 | 高 | 每个桥接函数小步验证；统一封装；开启 race/崩溃日志；优先使用 NSGo 已有封装 | 随机 crash、selector 不存在、对象提前释放 |
| R-05 | 线程 | 从后台 goroutine 直接操作 AppKit | 高 | 高 | 所有 UI 操作 dispatch 到 main queue；托盘菜单回调统一走 `onMain`；DB/剪贴板回调只发消息 | 非主线程断言、界面卡死、崩溃 |
| R-06 | 线程 | SQLite 写入阻塞主线程 | 中 | 中 | 单写 worker + WAL；读写分离；批量事务 | 搜索/滚动时卡顿 |
| R-07 | 数据 | SQLx migration 表与 Go migration 冲突 | 高 | 高 | 启动检测 `_sqlx_migrations`；先备份；把 `0001` 标记已应用；不改旧 migration | 重复建表、schema 不一致 |
| R-08 | 数据 | schema/列顺序/类型漂移 | 高 | 中 | 固定 schema 断言；旧 DB 副本测试；所有 SELECT/INSERT 显式列名 | `query_as` 空结果、字段错位 |
| R-09 | 数据 | `updated_at` 被错误刷新 | 高 | 高 | 仓储层区分“内容复用”和“元数据更新”；测试逐项断言 | 收藏/备注后排序变化 |
| R-10 | 数据 | `content_hash` 算法或输入拼接不一致 | 高 | 中 | 固定样本回归；与旧版逐字节对比 | 重复内容新增记录或错误合并 |
| R-11 | 数据 | 图片文件名/分片路径不一致 | 中 | 中 | 复用旧命名规则；旧数据只读验证 | 旧图片无法显示 |
| R-12 | 搜索 | FTS5 trigram 对 1–2 字符无效 | 中 | 高 | 保留 `LIKE` 兜底；中文按字符计数 | 短关键词搜索为空 |
| R-13 | 搜索 | FTS 查询语法/转义不一致 | 中 | 高 | 统一 query builder；特殊字符测试集 | 引号/星号/括号导致报错 |
| R-14 | 搜索 | `modernc.org/sqlite` 与 SQLx SQLite 行为差异 | 中 | 中 | 固定版本；兼容夹具；对比查询计划与结果 | 排序、collation、触发器差异 |
| R-15 | 剪贴板 | NSGo 内置剪贴板不支持 HTML/RTF/文件写入/监听 | 高 | 高 | 自建 `NSPasteboard` 服务；不依赖 `Scope` 基础能力 | 富文本粘贴丢失格式 |
| R-16 | 剪贴板 | 多格式写入被拆成多次 clear/write | 高 | 中 | 单个 `NSPasteboardItem` 一次写多表示 | 目标 App 只拿到纯文本 |
| R-17 | 剪贴板 | changeCount 轮询漏采/重采 | 高 | 中 | 120ms 轮询；记录 last changeCount；入库前去重 | 快速连续复制丢内容 |
| R-18 | 剪贴板 | 回环抑制失效导致自增记录 | 高 | 高 | 写回前登记记录内容及 plain/HTML/RTF 等所有实际表示 hash；TTL 2 秒；粘贴与复制共用 guard | 点击粘贴/复制后列表多一条 |
| R-19 | 剪贴板 | 来源应用抓取过晚 | 中 | 高 | 监听回调同步抓 frontmost app，再异步入库 | 来源显示为 EcoPaste/错误 App |
| R-20 | 剪贴板 | 图片解码/重编码改变像素或体积 | 中 | 中 | PNG 直通优先；非 PNG 明确转换规则；记录尺寸/字节 | 图片模糊、体积异常 |
| R-21 | 权限 | 辅助功能权限缺失导致粘贴失败 | 高 | 高 | 启动检测 `AXIsProcessTrustedWithOptions`；引导到系统设置；失败可重试 | 点击粘贴无反应 |
| R-22 | 权限 | 完全磁盘访问缺失导致旧数据/图标读取失败 | 中 | 中 | 检测 TCC 能力；给出可恢复提示；不静默失败 | 文件图标空白、导入失败 |
| R-23 | 时序 | 面板未让出 key 就发 ⌘V | 高 | 高 | hide/resign key 后延迟约 50ms；固定窗口粘贴后 restore key | 粘贴到错误 App 或无响应 |
| R-24 | 时序 | 失焦隐藏与编辑/菜单/文件面板冲突 | 中 | 高 | 已为系统保存面板接入原生隐藏暂停；继续统一“交互锁”，覆盖编辑、菜单、拖拽、固定窗口 | 备注编辑到一半窗口消失 |
| R-25 | UI | `NSTableView` 数据源/复用实现错误 | 高 | 中 | 小数据先行；固定行高/自适应行高分别验证；Instruments 检查 | 滚动错行、崩溃、内存涨 |
| R-26 | UI | 10k 条列表/缩略图导致卡顿 | 中 | 高 | 原生虚拟化；选择态仅刷新可视行（`rowsInRect:`）；追加分页复用已测量行高并用 `insertRowsAtIndexes` 增量插入（不整表 reload）；缩略图懒加载；图片缓存上限；分页 | 首屏 >300ms 或滚动掉帧 |
| R-27 | UI | 预览面板抢焦点或越界 | 中 | 中 | 非激活 panel；多屏坐标换算；边缘翻转 | 预览打开后无法继续输入 |
| R-28 | 拖拽 | `NSDraggingSession` 生命周期/多格式处理错误 | 中 | 中 | 文件/图片/文本分别验证；保留 128pt 预览图 | 拖到 Finder/浏览器失败 |
| R-29 | 图标 | `NSWorkspace` 图标提取慢或返回多表示 | 中 | 中 | 应用与文件图标均仅在首次提取，统一降采样到 128px 8-bit PNG，按文件类型/内容去重，启动与手动清理无引用文件 | 首次复制新来源应用或新扩展名文件时卡顿，或图标目录异常增长 |
| R-30 | 富文本 | HTML 导入触发远程资源加载或样式污染 | 中 | 中 | 不自动加载远程资源；限制 baseURL；只做展示/写回，不执行脚本 | 预览时发起网络请求 |
| R-31 | 备份 | Argon2 参数/算法名与旧容器不一致 | 高 | 中 | 按旧 header 读取；固定参数；round-trip 样本 | 旧加密备份无法导入 |
| R-32 | 备份 | AEAD nonce/header 长度/字节序错误 | 高 | 中 | 字节级契约测试；使用旧样本双向验证 | 解密失败或新包旧版打不开 |
| R-33 | 备份 | 大备份整包进内存导致 OOM | 中 | 中 | 流式 zip/加密；限制并发；进度回调 | 数 GB 资源导入崩溃 |
| R-34 | 备份 | merge 导入覆盖收藏/置顶/敏感标记 | 高 | 中 | 复用旧 merge 规则；字段级测试；先备份 | 导入后元数据丢失 |
| R-35 | 更新 | Tauri JSON endpoint 与 Sparkle appcast 不兼容 | 高 | 高 | 新增 appcast 生成；双端点并行；测试源验证 | 新版本检测不到或安装失败 |
| R-36 | 更新 | Sparkle EdDSA 签名/公钥配置错误 | 高 | 中 | 固定 key 管理流程；CI 校验；测试升级 | 签名校验失败 |
| R-37 | 发布 | Developer ID/公证/权限描述缺失 | 高 | 中 | 提前在测试机跑签名+公证；保留旧流水线 | Gatekeeper 拦截、TCC 弹窗异常 |
| R-38 | 发布 | Bundle ID 与旧版冲突导致数据串用 | 高 | 中 | 正式版同 ID；开发版独立 ID；启动校验数据根；`make dev`/`make build-dev` 已使用 `com.ayangweb.eco-paste.dev`，开发构建误用正式 ID 时退出；开发 bundle ID 优先映射 `dev` 数据根，`ECOPASTE_MODE=prod` 不能覆盖；曾因 `nsgo build` 无 `nsgodev` tag 误用正式数据根，已修复并用 `lsof` 验证；正式包登录项与升级路径待签名环境验收 | 开发版写进正式数据或改写正式 App 的登录项 |
| R-39 | 基线 | 旧工程归档后与迁移文档/夹具漂移 | 中 | 高 | 固定归档路径与 commit；文档记录归档位置；数据夹具独立保存 | 归档找不到或与文档不一致 |
| R-40 | 安全 | 日志记录完整剪贴板内容/密钥 | 高 | 中 | 统一脱敏 logger；禁止 debug 打印 content；日志审查 | 日志出现 token/私钥 |
| R-41 | 安全 | 备份密码/密钥进入仓库或 CI 日志 | 高 | 中 | 不提交样本密码；CI secret 注入；日志 mask | 仓库/日志出现密码 |
| R-42 | 窗口 | NSGo 面板每次显示都会自行定位，重定位依赖 `NSWindowDidOrderOnScreenNotification` 的私有名称 `_NSWindowDidBecomeVisible` | 中 | 低 | `viewDidMoveToWindow` 兜底；升级 macOS/NSGo 后回归三种打开位置 | 面板打开位置不再生效或闪跳 |
| R-42 | 设置 | JSON patch 合并语义与旧版不同 | 中 | 高 | 对象深合并、数组替换、跨字段校验；旧文件回环 | 保存后丢字段/数组被合并 |
| R-43 | 设置 | 默认值变化影响老用户 | 中 | 中 | 保持默认值；新增字段用零值兼容；迁移说明 | 升级后行为突变 |
| R-44 | 旧数据 | v0.6 导入字段映射错误 | 低 | 低 | 首版不提供 v0.6 数据目录导入，旧版用户走 `.ecopastebak` 备份恢复；保留旧版备份兼容测试 | 备份导入条目缺字段/重复 |
| R-45 | i18n | 托盘/菜单/命令文案遗漏 | 低 | 高 | 文案 key 清单；中英文逐项对比 | 切换语言后混排 |
| R-46 | 可访问性 | 自定义卡片/列表缺少 AX 信息 | 中 | 中 | 设置 label/role/help；VoiceOver 验收 | 读屏无法识别条目 |
| R-47 | 性能 | 启动时清理/图标/DB 迁移阻塞 | 中 | 中 | 迁移和清理放后台；首屏先可用；进度提示 | 冷启动 >2s |
| R-48 | 兼容 | 旧版仍写入时 NSGo 同时打开同一 DB | 高 | 中 | 双跑期不同数据目录；禁止同时写同一 DB；文件锁检测 | 出现 `database is locked` |
| R-49 | 许可 | 新依赖许可证/维护状态不合适 | 中 | 低 | 引入前评估 license、维护活跃度、跨平台；记录 ADR | 无法再分发或无人维护 |
| R-50 | 范围 | 提前做 Windows/Linux 抽象拖慢 macOS | 中 | 中 | 当前只实现 macOS；接口按现有需求最小化 | 出现无使用方的平台接口 |

## 3. 重点风险的处理方案

### 3.1 NSGo 版本固定

- `go.mod` 固定到已验证的正式稳定 tag，不使用 floating branch 或 `latest`。
- 升级 NSGo 时单独开 PR，且先按 `AGENTS.md` §4.1 阅读官方 CHANGELOG/API，评估新能力能否替代现有自研实现，必须通过：
  - `go test ./...`
  - `go test -race ./...`
  - `nsgo check`
  - 主面板/托盘/快捷键/粘贴人工回归
- 升级说明记录在 `docs/nsgo-migration/`，包括 API 变更和行为差异。

### 3.2 数据兼容

- 所有数据测试只使用真实数据副本，不直接操作生产目录。
- 首次启动 NSGo 版时：
  1. 检测数据根与 schema。
  2. 复制一份 DB 到临时目录做完整性检查。
  3. 通过后再写 migration 记录。
  4. 失败则进入只读提示，不继续写入。
- 任何 schema 变更都必须有：
  - 旧版可读的 migration 策略，或
  - 明确的版本门槛与备份/回滚说明。

### 3.3 剪贴板主路径

- 建立固定格式矩阵：text / html / rtf / png / jpeg / tiff / heic / files / 多格式并存 / 空内容 / 超大内容。
- 每次改动 `pasteboard`、`write`、`guard`、`keystroke` 都跑矩阵。
- 记录 macOS 版本与目标 App，因为不同 App 对 pasteboard 类型支持不同。

### 3.4 备份与更新

- 备份：旧版生成样本 → Go 读取 → Go 写出 → 旧版读取，四步都必须通过。
- 更新：测试 appcast 独立于生产；先验证 Sparkle 签名和安装，再切 stable。
- 发布：签名/公证在真实测试机验证，不以本地 `codesign -dv` 代替 Gatekeeper 验收。

## 4. 风险复盘节奏

| 时点 | 动作 |
| --- | --- |
| 每个 Phase 结束 | 更新本表状态，新增实际遇到的问题 |
| 每次数据层变更 | 复查 R-07 ~ R-14、R-31 ~ R-34 |
| 每次 UI/窗口变更 | 复查 R-05、R-23 ~ R-28 |
| 每次发布配置变更 | 复查 R-35 ~ R-39 |
| Cutover 前 | 全表逐项确认，未缓解项必须有明确接受人 |

## 5. 风险接受规则

- 高影响风险不允许“默认接受”，必须有缓解措施或明确延期理由。
- 接受风险时记录：接受人、接受范围、到期时间、回滚方案。
- 一旦触发条件出现，停止叠加新功能，先修复或回退。

## 6. 稳定性专项（2026-11）

针对置顶、搜索、点击条目、开关窗口等偶发闪退做的一次集中排查。R-04（purego/ObjC 内存）、R-05（后台 goroutine 操作 AppKit）与 R-25（NSTableView 复用）是实际命中的风险。

### 6.1 已确认的崩溃根因

| # | 现象 | 根因 | 归类 |
| --- | --- | --- | --- |
| 1 | 全局快捷键打开/隐藏面板时 `panic: core: the reactive graph belongs to another goroutine` | Carbon 热键回调在 AppKit 主线程直接调用 `Window.Show/Hide`（会 `Signal.Set`），不是 NSGo 反应式图的 goroutine；由 `Scope.Post` 投递即可（提交 `3602ef47`） | R-05 |
| 2 | 置顶、搜索、点击条目、开关面板等触发列表刷新时偶发 `SIGSEGV`，栈顶为 `setHistoryRowHovered.func1 ← transition.redrawView ← advanceTransitionFrames ← transitionTick` | 行悬停淡入淡出用 `rowHoverStates` 以 `objc.ID` 为键保存 `NSTableCellView` 裸指针。`reloadData`/`reloadDataForRowIndexes` 回收并释放 cell 后，共享 `NSTimer` 下一帧仍对已释放视图调用 `setNeedsDisplay:`，构成 use-after-free | R-04 / R-25 |

### 6.2 排查中发现的潜在崩溃隐患（同类问题）

| # | 隐患 | 根因 | 处理 |
| --- | --- | --- | --- |
| 3 | 已打开设置窗口时，从面板齿轮切换设置标签 | 图 goroutine 直接调用 `NSTabView` 的 `selectTabViewItemWithIdentifier:` 操作 AppKit | `SelectSettingsTab` 切标签改走 `onMain`，并跳过已关闭的窗口 |
| 4 | 设置页开关托盘图标 | `tray.SetIcon` 在设置保存的后台 goroutine 直接追加 NSGo 树的 mount item，与引擎 goroutine 竞争 | `settingsWiring.setTrayIcon` 改用 `app.Post` 回到图 goroutine |
| 5 | 反复开关设置窗口 | `settingsCaptureByTable` 与 apps/actions 不同，`SettingsView.Close` 未清理，重开窗口可能命中已释放视图 | `SettingsView.Close` 一并删除 capture 表映射 |
| 6 | 拖拽结束后关闭面板 | `checkDragFinished:` 延时选择器只取消 `activatePendingClick:`，未取消自身，可能投递到已释放的 `NSTableView` | `endPanelDrag` 取消 `checkDragFinished:` |

### 6.3 修复要点

- `transition` 增加被重绘视图字段 `view`：动画期间由共享驱动 `retain`，settle/`stop`/`snap` 时 `release`；新增 `newViewTransition(view)` 统一构造，所有视图型过渡（行悬停、图标按钮、导航胶囊、设置下拉/步进、信息按钮、输入框 chrome）均绑定视图。这样即使表视图在动画中途回收 cell，对象也不会被释放，最坏只是短暂多持有约 120ms。
- 列表 `reloadData` 前（`setPanelRows`/`setPanelQuery`/`setPanelDisplay`/`reloadPanelTable`）先 `clearRowHoverStates()`，避免对即将销毁的 cell 继续做淡出。
- 其余为上面 6.2 的线程与生命周期修正。

### 6.4 验证

- 自动：`gofmt -l .`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`nsgo check cmd/ecopaste` 全部通过。
- 新增回归测试（`internal/native/macos/transition_test.go`）：
  - `TestTransitionRetainsViewWhileAnimating`：动画期间驱动持有视图引用。
  - `TestAdvanceTransitionAfterOwnerRelease`：复现“列表先释放 cell、帧驱动再 tick”的崩溃序列，验证不再访问已释放视图且能收敛。
  - `TestClearRowHoverStatesReleasesViews`：刷新时清理悬停过渡并释放视图。
- 真机回归（人工，需覆盖高频/连续/快速切换）：置顶/取消置顶、长短词搜索、单击/双击/中键粘贴、收藏/删除/移动分组、反复开关面板、拖拽到 Finder、反复开关设置窗口并切换标签与托盘图标、深浅色。
