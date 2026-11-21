# 08 · 性能与生命周期排查（2026-11-21）

## 范围与原则

排查当前 Go / NSGo v0.2.0 / AppKit 应用的定时器、goroutine、窗口通知、鼠标监听、历史查询、图像处理及原生对象所有权。`legacy/` 不参与运行，不修改其中的 JavaScript 定时器；当前运行代码没有 `setInterval` / `setTimeout`。

只处理代码路径或回归测试能确认的重复工作和资源滞留。不引入额外调度框架，不升级依赖，不改变数据格式。收益应按具体路径理解，不能把局部基准等同于整个应用的速度提升。

## 已处理的问题

| 路径 | 确认的问题 | 调整与边界 |
| --- | --- | --- |
| 隐藏历史面板 | 启动及隐藏期间的剪贴板通知仍查询数据库、转换行、更新原生表格；显示/隐藏还人为发送数据变更通知 | 用 `desktop.CurrentWindow(cx).Visible()` 的响应式事件驱动加载；隐藏不刷新，重新打开读取最新页；删除开关窗口的人工数据通知。仍保留低成本的分组初始化 |
| 历史搜索 | 旧查询继续读库与转换内容，仅在提交结果时丢弃；搜索等待期间切换筛选会重复查询；分页可能混用新关键词与旧结果 | 每次加载使用可取消上下文；新查询、隐藏、销毁取消旧查询；立即保存最新输入；筛选/刷新消费已有 debounce；搜索等待期间不追加旧列表；相同关键词、分类或分组不重复触发 |
| 变更通知 | 多个通知可向图 goroutine 排队多个相同刷新；通知通道关闭后可能空转 | 排队中的刷新合并为一个，读取关闭通道立即退出；后续真实变更仍可刷新 |
| 行数据计算 | loading、预览等无关 props 更新也重新生成所有 `HistoryRow` | 使用 NSGo `Memo`，只在条目或语言变化时重新转换；不增加第二套缓存失效协议 |
| 暂停采集 | 120ms ticker 仍持续唤醒，只在 Poll 中跳过采集 | watcher 自己持有暂停状态，以容量 1 的事件通道通知配置变化；暂停停止 ticker，恢复重新创建；手动读取仍可用；已取消的 Poll 不读剪贴板 |
| Hover 监听 | 9 类自定义控件每次 `updateTrackingAreas` 都新增相同 `NSTrackingArea`，监听对象随布局次数累积 | 复用相同 owner / options 的 visible-rect tracking area，由 AppKit 更新几何；保留系统和其它用途的区域，生命周期仍由视图管理 |
| 原生所有权 | 设置 tabs、开关/下拉/快捷键、历史 table、hover bar、预览子视图与 footer divider 存在未配对创建引用；已释放 cell 的 Go layout 仍在全局映射 | 将所有权交给父视图后释放创建引用，或在 Close 配对释放；cell `dealloc` 清理 layout 与 hover 映射。保留动画期间的视图 retain 保护，避免为了释放而重新引入 use-after-free |
| 图片重复采集 | 已有按内容 hash 命名的缩略图，仍完整解码、缩放并编码一次，最后才检查文件已存在 | 先检查目标缩略图；命中直接返回，缺失仍重建；首次写入和未缓存的无效图片仍执行验证 |
| 点击延时 | 默认双击模式每次单击也安排单击延时任务，最终执行无效动作/刷新 | native props 明确传入启用的点击模式；只在启用单击动作时安排仲裁任务；隐藏、关闭、开始拖拽或禁用单击时取消等待；双击行为沿用原设置 |
| 拖拽结束 | 原生结束回调已收到后，兜底 delayed selector 仍可能再唤醒一次 | `finishDrag` 统一走 `endPanelDrag`，立即移除 monitor 并取消兜底任务；隐藏/关闭也清理。保留既有拖拽期间兜底，见下文验证边界 |
| 异步操作退出 | 备份、托盘手动读取、设置中的数据库/数据目录操作从 `context.Background()` 派生，应用销毁后仍可能等到自身超时 | runtime 在图 goroutine 初始化时捕获应用 Scope 上下文，后台操作统一从该上下文派生，保留各操作的上限；后端已有取消检查继续负责中断 IO |

修改数字输入控件的 tracking 时，将超过 600 行文件内的 target 注册部分等量移到 `settings_number_targets.go`；没有改变数值行为。复核时按同一文件长度规范，将面板既有的选择/导航逻辑等量移到 `panel_selection.go`，没有修改选择规则。

### 窗口可见状态的依据

实际冷启动回归发现，既有 `_NSWindowDidBecomeVisible` 通知不能可靠承担“首次加载数据”的职责。数据加载因此改用 NSGo v0.2.0 的公开 `CurrentWindow().Visible()` 信号，覆盖程序 Show/Hide、托盘 Toggle 等路径；不再通过原生 `opened` 通知重复刷新。原生显示通知继续只承担既有定位、动画与导航同步。

可见状态转为隐藏时同时向 native view 投递 `cancelInteractions`，取消点击/拖拽等待并关闭 hover 状态；资源清理不依赖私有窗口通知是否送达。

`Window.Show/Hide` 自身写入可见信号并排队窗口命令，热键通过既有 `Scope.Post` 回到图 goroutine，引擎处理 posted work 后提交各 surface；不需要靠剪贴板数据通知来刷新窗口命令。回归测试直接创建 NSGo panel，验证无原生通知时首次显示仍加载、隐藏取消、再次显示重查。

## 保留的定时任务

| 任务 | 调度 | 保留原因 / 释放方式 |
| --- | --- | --- |
| 剪贴板变化计数 | 活跃时 120ms | 仅检测 changeCount，未变化不读内容；暂停停止 ticker，退出停止循环 |
| 历史自动清理 | 启用限制时每 15 分钟 | 时间过期本身没有用户事件；无有效限制不创建 timer；规则变化事件驱动，退出取消并等待结束 |
| 日志保留清理 | 每小时 | 长时间运行时清除过期归档；logger Close 停止 ticker 并等待 worker |
| 统一交互过渡 | 动画期间 60Hz | 全应用共享单个 NSTimer；所有过渡收敛/停止后 invalidate，不为空闲控件持续重绘 |
| 搜索 debounce | 120ms 单次 | 合并连续输入；替换、主动刷新、隐藏、销毁均取消 |
| 粘贴焦点交接 | 70ms；固定窗口恢复 key 为 50ms | 给目标 App 接收焦点和粘贴事件的时间；Scope 持有单次任务，销毁取消 |
| 单击/双击仲裁 | 系统 doubleClickInterval + 50ms | 只有启用单击动作时创建，避免一次双击触发两次动作；相关生命周期及时取消 |
| 拖拽兜底 | 开始后 200ms，鼠标仍按下时 100ms | 只在真实拖拽期间运行；原生结束回调、隐藏、关闭立即取消。此次自动输入未能确认真实拖放，暂不删除曾参与失焦隐藏修复的保障逻辑 |
| 模态弹窗抬升 | 50ms 单次 | `NSAlert.runModal` 会重置预先设置的 level，需在 modal runloop 内再抬升；非循环任务 |
| watcher 退出等待 | 最多 150ms | 限制退出等待，不是后台轮询 |
| 单实例 PID 读取重试 | 最多 5 次 × 50ms | 锁已获取但 PID 尚未写入的启动竞争窗口；有明确次数上限 |
| context deadlines | 5s–10min，按操作 | 限定数据库/手动采集/备份等单次操作，不是周期任务；均在函数返回时 cancel |

没有为降低唤醒频率而延长活跃剪贴板采样间隔、改动粘贴时序或移除交互动画。系统 Sparkle / AppKit 自行管理的内部调度不复制实现。

## 可复现基准

Apple M1 Pro，重复存储相同的 1600×1000 PNG，预先写入原图与缩略图，3 轮、每轮 5 次操作：

```sh
go test ./internal/storage -run '^$' -bench BenchmarkImageStoreRepeatedPNG -benchmem -benchtime=5x -count=3
```

| | 修改前 | 修改后 |
| --- | --- | --- |
| 每次耗时 | 29.25–30.24ms | 78.2–84.8µs |
| 每次分配 | 约 17.64MB | 1,440–1,456B |
| 每次分配次数 | 72–73 | 10 |

这衡量的是缩略图缓存命中的重复采集路径；未证明首次解码、更大历史库、整应用 CPU 或总内存具有相同比例的改进。

文件历史行转换的缺失路径回退曾对每个缺失项重复拆分一次完整的 `file_types` 字符串。现在只在首次需要回退时解析一次。Apple M1 Pro、64 个缺失路径、2 秒基准、3 轮：

```sh
go test ./internal/app -run '^$' -bench '^BenchmarkDescribeFileEntriesMissing$' -benchmem -benchtime=2s -count=3
```

| | 修改前 | 修改后 |
| --- | --- | --- |
| 每次耗时 | 150.7–156.0µs | 95.5–101.8µs |
| 每次分配 | 102,784B | 30,080–30,144B |
| 每次分配次数 | 257 | 194 |

此基准只衡量 64 个路径均缺失且需使用文件类型提示的转换，不代表实际每页的平均收益。

托盘存储用量原先分别遍历数据库、资源、配置和整个数据根目录，因此这些子目录被重复枚举。现在由一次根目录遍历累加分项与总量。Apple M1 Pro、资源目录内 1,000 个 1KiB 文件、2 秒基准、3 轮：

```sh
go test ./internal/storage -run '^$' -bench '^BenchmarkMeasureUsage$' -benchmem -benchtime=2s -count=3
```

| | 修改前 | 修改后 |
| --- | --- | --- |
| 每次耗时 | 6.61–6.69ms | 3.30–3.42ms |
| 每次分配 | 1,519,476–1,519,478B | 762,605–762,613B |
| 每次分配次数 | 12,174 | 6,108 |

此基准反映资源目录文件较多时手动请求存储统计的路径，不代表启动时或日常剪贴板路径。

备份合并原先为每条历史记录单独执行一次存在性查询，再执行插入。现在在事务内复用预编译语句，由同一条 `INSERT ... SELECT ... WHERE NOT EXISTS` 语句检查 `(kind, content_hash)` 并插入。Apple M1 Pro、1,000 条文本记录、5 次操作、3 轮：

```sh
go test ./internal/database -run '^$' -bench '^BenchmarkMergeBackupItems$' -benchmem -benchtime=5x -count=3
```

| | 修改前 | 修改后 |
| --- | --- | --- |
| 每次耗时 | 97.1–99.7ms | 77.2–80.0ms |
| 每次分配 | 5,057,891–5,073,651B | 4,143,852–4,144,089B |
| 每次分配次数 | 108,165–108,196 | 87,962–87,963 |

合并语义保持不变：已有相同 kind/hash 的记录计入 `Skipped`，其余记录保留备份顺序插入。

## 验证

- `history_loading_test.go`：隐藏不查询、窗口信号驱动加载、取消旧请求、隐藏保留最新搜索、销毁取消、筛选消费 debounce、禁止混合分页、行 memo。
- `watcher_lifecycle_test.go`：暂停不再读取 changeCount、恢复继续采集、暂停时手动读取、取消后不读源。
- `tracking_test.go`：9 类控件各更新 100 次只保留一个匹配区域，并保留无关区域。
- `lifecycle_test.go`：关闭设置/面板后代表性原生子控件没有遗留创建引用；连续创建释放 100 个 cell 不遗留 Go layout。
- `clipboard_runtime_lifecycle_test.go`：操作类和返回值类的超时包装都服从 owner 取消。
- 图片测试：已有缩略图不重算、缺失时重建、首次无效数据仍拒绝；既有 Go / race / NSGo 检查覆盖相关模块。
- 真机开发包使用 `com.ayangweb.eco-paste.dev`；在独立 `ECOPASTE_HOME` 的 150 条临时夹具上验证首次加载、快速替换搜索最终结果、Space/Esc 预览、打开设置、数字控件 3→4→3，以及关闭后重新打开设置。

自动检查：`git diff --check`、`gofmt -l .`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`nsgo check cmd/ecopaste`、`nsgo test cmd/ecopaste`、`make build-dev`。其中 `nsgo test` 没有独立应用测试用例（0 passed / 0 failed / 0 skipped），不能算作额外 UI 覆盖。

`internal/ui`、`internal/clipboard`、`internal/native/macos`、`internal/app` 另以 `-count=10` 连续通过。复核时清理了此前记录的 4 个原生包静态检查项：合并 hover transition 声明，并移除未使用的 associated-object key、搜索框 class 缓存和 settings reset 字段；这些字段均无运行时引用。

验证边界：自动键盘事件未能确认 Carbon 全局热键，自动 drag 未能确认真实的接受/取消拖放；未进行物理键鼠、目标应用实际粘贴、长时间 Instruments CPU / Allocations / 能耗剖析或完整发布验收。上述入口应做人工主路径回归，当前不宣称所有性能与稳定性风险已消除。

## 中断后的复核与补充（2026-11-21）

逐项核对未提交 diff、上述修复路径及回归测试，补齐两处遗漏：

- `Scope.Context()` 返回当前执行 owner 的上下文，并非固定返回接收者 Scope 的上下文。设置操作由 worker 调用时不能读取它，否则会访问图的可变状态，并可能绑定到临时 computation。现在 runtime 初始化时捕获上下文；设置、备份和托盘手动读取复用它。单实例信号与激活请求循环也在启动 goroutine 前捕获上下文。`TestSettingsWorkerUsesCapturedRuntimeContext` 验证设置 worker 无需访问图，并服从 runtime 取消；原有操作/返回值两类超时取消测试继续通过。
- 改为可见状态加载后，model 不可用的首次显示会提前返回，导致一直显示 loading。现在仍通过统一加载入口显示数据库不可用错误；`TestOpeningHistoryWithoutServiceShowsError` 覆盖此分支。

补充 `TestNativeDragEndCancelsFallback` 与 `TestPanelCancelsQueuedClickActions`，实际安排原生 delayed selector，验证原生拖拽结束、隐藏、禁用单击动作后延时任务立即释放其 table 引用；不将此测试等同于真实拖放成功。

本轮复核通过 `staticcheck ./...`，此前记录的 4 个原生包告警已清除。完整测试及打包结果以本次清理后的验证为准；`nsgo test` 没有独立应用测试用例，不能算作额外 UI 覆盖。

同一重复 PNG 基准复测为 77.3–82.8µs/op、1,424–1,456B/op、10 allocs/op。以新建的独立 `ECOPASTE_HOME` 和 `EcoPaste-2.0.0-macos_arm64/EcoPaste.app` 验证 150 条临时历史的冷启动加载，alpha 替换为 beta 后仅有 75 条匹配结果，Space/Esc 打开/关闭对应预览，数字控件 3→4→3，以及设置关闭后重开。只使用临时夹具，未操作正式数据；真实拖放、物理全局热键及目标应用粘贴仍保留上述验证边界。
