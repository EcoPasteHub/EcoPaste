# TODO: CI/CD 发布流水线

## 目标

建立 GitHub Actions 发布流水线，tag `v*` 触发 macOS 与 Windows 多架构构建，并创建 Release。

## 范围

- 新增或完善 `release.yml`。
- 构建矩阵：
  - macOS arm64 / x64。
  - Windows x64 / arm64。
- 缓存 Rust、pnpm 与 Tauri 构建依赖。
- 生成 changelog 并创建 GitHub Release。
- macOS 签名/公证。
- Windows NSIS 安装包。

## 注意

- secrets 命名要和本地文档一致。
- 自动更新签名密钥与 CI secrets 联动。
- Portable 版本如果尚未完成，release workflow 先预留产物位，不强行产出。

## 验收

- 推送测试 tag 后能产出可下载安装包。
- Release notes 自动生成且可手工编辑。
- 失败时日志足够定位 Rust、pnpm、签名或公证问题。
- 不在 CI 中提交生成文件。

