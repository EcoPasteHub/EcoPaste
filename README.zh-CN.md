<div align="center">
  <img src="./public/logo.png" alt="EcoPaste" width="96" height="96" />

  # EcoPaste

  **适用于 macOS 与 Windows 的本地优先剪贴板管理器。**

  [English](./README.md) | 简体中文

  <br />

  <img alt="Tauri v2" src="https://img.shields.io/badge/Tauri-v2-24c8db?style=flat-square" />
  <img alt="Rust first" src="https://img.shields.io/badge/Rust-first-b7410e?style=flat-square" />
  <img alt="React 19" src="https://img.shields.io/badge/React-19-61dafb?style=flat-square" />
  <img alt="macOS" src="https://img.shields.io/badge/macOS-supported-000000?style=flat-square&logo=apple&logoColor=white" />
  <img alt="Windows" src="https://img.shields.io/badge/Windows-supported-0078d4?style=flat-square&logo=windows&logoColor=white" />
</div>

## 关于

EcoPaste 是一个开源桌面剪贴板管理器，采用 Rust-First 的 Tauri 架构：持久化和系统侧能力优先由 Rust 承担，React 前端专注于界面展示与交互。

EcoPaste 的目标是更快、更轻、更易维护，并提供本地存储、SQLite 搜索、原生快捷键、托盘、备份，以及聚焦 macOS 与 Windows 的跨平台体验。

## 功能

- 采集纯文本、HTML、RTF、图片、文件和文件夹等剪贴板内容。
- 使用 SQLite FTS5 搜索剪贴板正文与备注。
- 按来源应用和内容类型过滤历史记录。
- 识别并跳过高置信敏感内容，例如私钥、服务 Token、AWS Key 和 JWT。
- 在独立预览窗口中查看文本、图片和文件记录。
- 支持粘贴、复制、复制为纯文本、定位文件、打开链接、添加备注、置顶、收藏、删除，以及将记录拖出到其它应用。
- 通过收藏、置顶、备注、自定义分组和可配置快捷动作组织历史记录。
- 可调整采集顺序、大小限制、保留策略、展示密度、列表排序和窗口行为。
- 支持导出和导入 `.ecopastebak` 备份，包括加密备份包。
- 剪贴板数据、资源缓存和设置均保存在本机。

## 参与贡献

开发环境、架构说明、质量检查和贡献要求请阅读[贡献指南](./CONTRIBUTING.zh-CN.md)。

## 开源协议

EcoPaste 基于 [Apache License 2.0](./LICENSE) 开源。
