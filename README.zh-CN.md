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

EcoPaste 是一个开源桌面剪贴板管理器。本仓库是 EcoPaste 的 Rust-First 重构版本：持久化和系统侧能力优先由 Rust 承担，React 前端专注于界面展示与交互。

这次重构的目标是让应用更快、更轻、更易维护，并提供本地存储、SQLite 搜索、原生快捷键、托盘、备份，以及聚焦 macOS 与 Windows 的跨平台体验。

## 项目状态

当前仓库仍处于 beta 重构阶段（`0.6.0-beta.3`），还不是稳定发布通道。

尝试这个版本前，请先备份旧版 EcoPaste 的重要数据。重构版调整了运行架构、设置模型、存储布局和数据库结构；在 beta 阶段，不保证与旧版数据兼容。

## 平台范围

Rust-First 重构版仅支持 macOS 与 Windows。

旧版 EcoPaste 曾支持 Linux，但重构版已经放弃 Linux 支持，并且暂不计划重新支持 Linux。如果你需要 Linux 支持，请继续使用旧版发布线。

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

## 架构

EcoPaste 采用 Rust-First 的 Tauri 架构：

- `src-tauri/src/clipboard/` 负责剪贴板采集、内容识别、写回、来源应用、资源落盘和监听回环抑制。
- `src-tauri/src/db/` 负责 SQLite 仓储、模型、迁移和 FTS 搜索。
- `src-tauri/src/settings/`、`window/`、`shortcut/`、`tray/`、`menu/`、`autostart/` 和 `backup/` 负责原生能力与持久化应用状态。
- `src/` 包含 React UI、Ant Design 组件、UnoCSS 样式、Valtio UI/设置镜像、i18n 资源，以及类型化 Tauri command 封装。

前端通过 Tauri command 调用 Rust，并通过 `clipboard://updated`、`settings://updated`、`window://visibility` 等命名空间事件接收刷新信号。

## 技术栈

| 维度 | 选型 |
| --- | --- |
| 桌面外壳 | Tauri v2 |
| 前端 | React 19、Ant Design 6、UnoCSS `presetWind4` |
| 状态 | Valtio，仅用于 UI 状态与设置镜像 |
| 后端 | Rust、sqlx、SQLite |
| 构建 | Vite、pnpm |
| 质量 | Biome、TypeScript、rustfmt、clippy、cargo test |

## 开始开发

### 环境要求

- macOS 或 Windows。
- Node.js 20 或更高版本。
- pnpm 10 或更高版本。
- `rust-toolchain.toml` 指定的 Rust 工具链（`1.96.0`，包含 `rustfmt` 和 `clippy`）。
- Tauri v2 所需的系统原生依赖。请参考 [Tauri prerequisites](https://tauri.app/start/prerequisites/) 中对应系统的说明。

### 安装依赖

```bash
pnpm install
```

### 开发运行

```bash
pnpm tauri dev
```

### 构建

```bash
pnpm tauri build
```

## 质量检查

前端：

```bash
pnpm lint
pnpm tsc
```

Rust：

```bash
cd src-tauri
cargo fmt
cargo clippy -- -D warnings
cargo test
```

格式化前端文件：

```bash
pnpm format
```

## 仓库结构

```text
src-tauri/
  src/
    commands/    # Tauri command 入口
    clipboard/   # 剪贴板读写、采集、识别、存储
    db/          # SQLite 仓储、模型、迁移
    settings/    # 设置模型与持久化
    window/      # 窗口状态、定位、生命周期
    shortcut/    # 全局快捷键
    tray/        # 托盘菜单
    menu/        # 列表项右键菜单
    backup/      # 备份导入导出
    i18n/        # Rust 侧用户可见文案
  migrations/
src/
  commands/      # 类型化 Tauri invoke 封装
  components/    # 共享 React 组件
  constants/     # 跨层复用常量镜像
  hooks/         # 共享 hooks
  locales/       # zh-CN 和 en-US 翻译
  pages/         # Clipboard、Preference、Preview、ContextMenu
  stores/        # Valtio UI 状态与设置镜像
  types/         # TypeScript 契约镜像
```

## 参与贡献

修改代码前请先阅读 [AGENTS.md](./AGENTS.md)。它是本重构版架构边界、平台范围、编码规范和质量要求的单一真相源。

涉及下个版本的用户可见能力变更时，请同步更新 [RELEASE-NEXT.md](./RELEASE-NEXT.md)。文档内容也应与当前 beta 状态和受支持平台保持一致。
