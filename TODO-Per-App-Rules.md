# TODO: 每应用规则

## 目标

支持按来源应用配置独立行为规则，例如在某个应用中强制纯文本粘贴、禁用 EcoPaste、仅不捕获复制内容。

## 规则模型

- `Settings.clipboard.appRules: AppRule[]`
- `AppRule` 字段：
  - `appId`
  - `appName`
  - `behavior`
- `behavior`：
  - `forcePlainPaste`
  - `disableAll`
  - `ignoreCapture`

## Rust 命中时机

- 捕获链路：命中 `ignoreCapture` 或 `disableAll` 时不入库。
- 粘贴链路：命中 `forcePlainPaste` 时强制 `plain = true`；命中 `disableAll` 时拒绝操作。
- 唤窗/快捷键链路：命中 `disableAll` 时拒绝打开主窗。

## 平台采集

- macOS 复用 `NSWorkspace.frontmostApplication`。
- Windows 补齐 `GetForegroundWindow`、`GetWindowThreadProcessId`、`QueryFullProcessImageNameW`，主键使用小写 exe 名或稳定路径策略。
- 平台差异封在 Rust 模块内部。

## 前端设置页

- 在偏好设置的剪贴板相关区域新增“应用规则”。
- 列表展示应用名、应用标识、行为和删除入口。
- 新增规则来源：
  - 最近来源应用。
  - 当前前台应用。
- 命中 `disableAll` 后通过事件或命令错误给用户提示。

## 验收

- 不同应用下捕获、粘贴和唤窗行为都按规则生效。
- 全局 `copyPlain` / `pastePlain` 仍作为默认值，应用规则优先级更高。
- `disableAll` 优先级最高。
- 规则修改实时生效，无需重启。

