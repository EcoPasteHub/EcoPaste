# 参与贡献

[English](./CONTRIBUTING.md) | 简体中文

感谢你帮助改进 EcoPaste。本指南说明本项目的范围、开发环境、架构边界、质量检查，以及代码和文档贡献的基本要求。

## 项目状态

EcoPaste 已进入正式发布通道。后续变更应直接演进当前应用；涉及已发布用户数据的存储、设置或数据库契约变化时，需要提供
migration 或升级处理。

## 平台范围

EcoPaste 仅支持 macOS 与 Windows。

Linux 不在支持范围内，新增代码、依赖、构建产物和文档都应聚焦 macOS 与 Windows。

## 修改代码前

请先阅读[AGENTS.md](./AGENTS.md)。它是本仓库架构边界、平台范围、编码规范和质量要求的单一真相源。

请尊重当前工作区状态，不要覆盖或回滚并非由你产生的改动。

提交信息使用单行 Conventional Commits，例如 `feat:`、`fix:`、`refactor:` 或 `docs:`。

## 架构

EcoPaste 采用 Rust-First 的 Tauri 架构：

- `src-tauri/src/clipboard/` 负责剪贴板采集、内容识别、写回、来源应用、资源落盘和监听回环抑制。
- `src-tauri/src/db/` 负责 SQLite 仓储、模型、迁移和 FTS 搜索。
- `src-tauri/src/settings/`、`window/`、`shortcut/`、`tray/`、`menu/`、`autostart/` 和
  `backup/` 负责原生能力与持久化应用状态。
- `src/` 包含 React UI、Ant Design 组件、UnoCSS 样式、Valtio UI/设置镜像、i18n 资源，以及类型化
  Tauri command 封装。

前端通过 Tauri command 调用 Rust，并通过 `clipboard://updated`、`settings://updated`、`window://visibility`
等命名空间事件接收刷新信号。

## 技术栈

| 维度     | 选型                                           |
| -------- | ---------------------------------------------- |
| 桌面外壳 | Tauri v2                                       |
| 前端     | React 19、Ant Design 6、UnoCSS `presetWind4`   |
| 状态     | Valtio，仅用于 UI 状态与设置镜像               |
| 后端     | Rust、sqlx、SQLite                             |
| 构建     | Vite、pnpm                                     |
| 质量     | Biome、TypeScript、rustfmt、clippy、cargo test |

## 开始开发

### 环境要求

- macOS 或 Windows。
- Node.js 20 或更高版本。
- pnpm 10 或更高版本。
- `rust-toolchain.toml` 指定的 Rust 工具链（`1.96.0`，包含 `rustfmt` 和 `clippy`）。
- Tauri v2 所需的系统原生依赖。请参考
  [Tauri prerequisites](https://tauri.app/start/prerequisites/) 中对应系统的说明。
- Trellis 工作流文档请参考 [Trellis 文档](https://docs.trytrellis.app/zh)。

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

## 贡献检查清单

- 除非现有架构明确要求放在其它层，业务逻辑、原生能力、数据库访问、存储、设置持久化和平台集成都应放在 Rust。
- React 侧专注于渲染、交互、UI 状态、前端 i18n 和预览。
- command 名、事件名、channel、storage key 等跨层复用常量需要同时维护 Rust 常量和 `src/constants/` 镜像。
- 已发布 schema 变更必须新增 migration，不要直接修改已发布 migration。
- 前端用户可见文案需要同步更新 `zh-CN` 和 `en-US` 语言资源。
- 托盘、原生菜单、命令返回 toast 等 Rust 侧短文案走 `i18n/`。
- 针对改动范围运行检查；触及共享行为或跨层契约时，需要扩大验证范围。
