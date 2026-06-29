# Auto Updater

## Goal

接入 Tauri v2 自动更新能力，支持检查更新、下载、安装提示与签名校验。

## Scope

- 引入 `tauri-plugin-updater`。
- 配置更新端点。
- 配置签名密钥环境变量 `TAURI_SIGNING_PRIVATE_KEY`。
- 在偏好设置或关于页提供检查更新 UI。
- 提供独立“软件更新”窗口，承载检查中、可更新、下载中、已是最新、错误等状态。
- 让设置项 `update.autoCheck` 与 `update.includeBeta` 和更新逻辑联动。

## Implementation Notes

- 按官方 Tauri v2 updater 文档配置 plugin、capability 和 `tauri.conf`。
- 更新端点区分稳定版与 beta 版。
- 下载、安装和重启路径需要清晰日志。
- 关于页/偏好设置提供入口打开独立软件更新窗口，窗口展示当前版本、更新状态和错误；release notes 元数据保留在后端响应中，当前 UI 暂不展示。
- 自动检查在启动后一段延迟执行，避免拖慢首屏。
- 失败信息通过统一命令错误处理与日志输出。

## Acceptance Criteria

- 本地 mock 更新端点可检测到新版本。
- 签名不匹配时拒绝安装。
- `includeBeta` 关闭时不提示 beta 包。
- 用户可从偏好设置/关于页手动打开独立软件更新窗口并执行检查、下载、安装重启或跳过当前版本。
- macOS 与 Windows 打包产物都能走同一更新流程。
