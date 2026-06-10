# TODO: 首次启动引导

## 目标

首次启动时显示独立引导窗口，分步完成关键设置：欢迎、权限、快捷键、忽略应用、旧版数据导入和完成页。

## 数据模型

- `Settings.onboarding.completed: bool`，默认 false。
- `Settings.onboarding.lastStep: number`，用于中途关闭后恢复。
- 旧版导入状态可放在 `Settings.onboarding.legacyImport` 下，记录是否检测、是否导入、导入类型和时间。

## Rust

- 新窗口 label：`onboarding`，URL `/#/onboarding`。
- 启动时根据 `SettingsStore.snapshot().onboarding.completed` 决定是否创建。
- 命令：
  - `open_onboarding()`
  - `set_onboarding_step(step)`
  - `finish_onboarding()`
  - `check_permissions()`
  - `detect_legacy_data()`
  - `import_legacy_data(types)`
  - `cancel_legacy_import(task_id)`
- macOS 权限检测至少覆盖辅助功能；屏幕录制、输入监控可按真实需求决定是否展示。
- 旧版数据路径实现前先核对本地 `EcoPaste_bak` 与旧版 identifier。

## 前端

- 新增 `pages/Onboarding`。
- 使用 Ant Design 组件或项目内现有基础控件实现 stepper。
- 每一步独立组件，步骤数组集中定义，便于后续追加步骤。
- 快捷键步骤复用设置页快捷键录入控件。
- 忽略应用步骤复用近期来源应用和设置写回能力。
- 旧版导入步骤显示类型数量、总大小、进度和取消按钮。

## 旧版导入

- 用只读连接打开旧 DB，不触发迁移。
- 按用户选择类型转换为新 schema。
- 图片复制到当前资源目录并重算 hash。
- 走现有入库和去重逻辑。
- 长任务用后台 task + 进度事件，不阻塞窗口。

## 验收

- 首次启动自动弹出，引导完成后不再自动弹。
- 中途关闭后下次从上次步骤继续。
- 设置页可重新打开引导。
- 旧版导入可取消，失败不会留下半截数据。
- macOS 与 Windows 都有合理降级路径。

