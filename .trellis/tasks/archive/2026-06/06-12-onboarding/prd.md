# Onboarding

## Goal

首次启动时显示独立引导窗口，分步完成关键设置：欢迎、权限、快捷键、忽略应用、旧版数据导入和完成页。

## Scope

- 引导 UI 参考 HapiGo 的深色无边框设置向导：大标题、中心化内容、底部主按钮、权限截图式说明、完成页。
- 引导窗口始终使用 dark 模式，不跟随全局主题。
- 引导窗口设置 `decorations: false`，固定尺寸并在打开时居中显示。
- macOS 引导窗口用 CSS `rounded-4` 做圆角；Windows 保持普通无边框矩形。
- 右上角语言切换使用下拉菜单，不使用分段按钮。
- 引导未完成前不提供关闭按钮，并拦截原生关闭请求，属于强制引导。
- 新增 `Settings.onboarding.completed: bool`，默认 false。
- 新增 `Settings.onboarding.lastStep: number`，用于中途关闭后恢复。
- 旧版导入状态可放在 `Settings.onboarding.legacyImport` 下，记录是否检测、是否导入、导入类型和时间。
- 新窗口 label 为 `onboarding`，URL 为 `/#/onboarding`。
- 启动时根据 `SettingsStore.snapshot().onboarding.completed` 决定是否创建。
- 新增 `pages/Onboarding`。
- 引导步骤拆分为独立组件，步骤数组集中定义；能用 Ant Design 组件实现的交互优先使用 Ant Design，不重复造基础控件。

## Implementation Notes

- Rust 命令包括 `open_onboarding()`、`set_onboarding_step(step)`、`finish_onboarding()`、`check_permissions()`、`detect_legacy_data()`、`import_legacy_data(types)`、`cancel_legacy_import(task_id)`。
- macOS 权限检测至少覆盖辅助功能；屏幕录制、输入监控可按真实需求决定是否展示。
- 旧版数据路径实现前先核对本地 `EcoPaste_bak` 与旧版 identifier。
- 使用 Ant Design 组件或项目内现有基础控件实现 stepper。
- 每一步独立组件，步骤数组集中定义，便于后续追加步骤。
- 快捷键步骤复用设置页快捷键录入控件。
- 忽略应用步骤复用近期来源应用和设置写回能力。
- 旧版导入步骤显示类型数量、总大小、进度和取消按钮。
- 旧版导入用只读连接打开旧 DB，不触发迁移。
- 按用户选择类型转换为新 schema；图片复制到当前资源目录并重算 hash。
- 导入走现有入库和去重逻辑。
- 长任务用后台 task + 进度事件，不阻塞窗口。

## Acceptance Criteria

- 首次启动自动弹出，引导完成后不再自动弹。
- 引导未完成时不能通过关闭按钮或原生关闭请求隐藏/销毁窗口。
- 中途异常退出后下次从上次步骤继续。
- 设置页可重新打开引导。
- 旧版导入可取消，失败不会留下半截数据。
- macOS 与 Windows 都有合理降级路径。
