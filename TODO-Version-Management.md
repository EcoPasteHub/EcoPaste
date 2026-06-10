# TODO: 版本管理与 release-it

## 目标

配置统一版本发布命令，支持正式版、RC 和 beta，保证 `package.json`、Tauri 配置和发布 tag 一致。

## 范围

- 接入 `release-it` 或等价发布工具。
- 命令：
  - `release`
  - `release-rc`
  - `release-beta`
- Conventional Commits 驱动版本号与 changelog。
- tag 命名与 CI 触发规则一致。

## 注意

- 当前 `package.json` 版本 `0.6.0-beta.3` 是项目未发版开发期边界；一旦改成其他版本，数据兼容策略也要随之改变。
- 版本变更前要确认数据库 schema 与迁移策略。

## 验收

- dry-run 能输出正确版本、tag 和 changelog。
- 正式版、RC、beta 的 npm/package metadata 与 Tauri bundle version 一致。
- CI 能被生成 tag 正确触发。

