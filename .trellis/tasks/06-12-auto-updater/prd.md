# Auto Updater

## Goal

接入 Tauri v2 自动更新能力，支持检查更新、下载、安装提示与签名校验。

## Scope

- 引入 `tauri-plugin-updater`。
- 配置更新端点。
- 配置签名密钥环境变量 `TAURI_SIGNING_PRIVATE_KEY`。
- 在偏好设置或关于页提供检查更新 UI。
- 让设置项 `update.autoCheck` 与 `update.includeBeta` 和更新逻辑联动。

## Implementation Notes

- 按官方 Tauri v2 updater 文档配置 plugin、capability 和 `tauri.conf`。
- 更新端点区分稳定版与 beta 版。
- 下载、安装和重启路径需要清晰日志。
- 关于页展示检查更新按钮、当前版本、更新状态与错误。
- 自动检查在启动后一段延迟执行，避免拖慢首屏。
- 失败信息通过统一命令错误处理与日志输出。

## Acceptance Criteria

- 本地 mock 更新端点可检测到新版本。
- 签名不匹配时拒绝安装。
- `includeBeta` 关闭时不提示 beta 包。
- macOS 与 Windows 打包产物都能走同一更新流程。
