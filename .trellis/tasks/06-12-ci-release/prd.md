# CI/CD Release Pipeline

## Goal

建立 GitHub Actions 发布流水线，tag `v*` 触发 macOS 与 Windows 多架构构建，并创建 Release。

## Scope

- 新增或完善 `release.yml`。
- 构建矩阵覆盖 macOS arm64 / x64、Windows x64 / arm64。
- 缓存 Rust、pnpm 与 Tauri 构建依赖。
- 生成 changelog 并创建 GitHub Release。
- 支持 macOS 签名 / 公证。
- 产出 Windows NSIS 安装包。

## Implementation Notes

- secrets 命名要和本地文档一致。
- 自动更新签名密钥与 CI secrets 联动。
- Portable 版本如果尚未完成，release workflow 先预留产物位，不强行产出。
- CI 不应提交生成文件。

## Acceptance Criteria

- 推送测试 tag 后能产出可下载安装包。
- Release notes 自动生成且可手工编辑。
- 失败日志足够定位 Rust、pnpm、签名或公证问题。
- CI 流程不会修改并提交仓库文件。
