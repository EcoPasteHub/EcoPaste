# 参与贡献

[English](./CONTRIBUTING.md) | 简体中文

请先阅读 [README.zh-CN.md](./README.zh-CN.md)，了解项目概览和平台范围。随后阅读 [AGENTS.md](./AGENTS.md)，它是 EcoPaste Rust-First 架构边界、编码规范、发布说明策略和支持平台的单一真相源。

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

## 普通开发

如果你不使用 AI 编码工具，请按以下流程贡献：

1. 从当前目标分支创建开发分支。
2. 按本文说明完成环境配置、运行、构建和质量检查。
3. 按 [AGENTS.md](./AGENTS.md) 的要求，将持久化和系统侧能力放在 Rust，将界面展示与交互放在 React。
4. 新增用户可见能力时，同步更新 [RELEASE-NEXT.md](./RELEASE-NEXT.md)。
5. 提交 PR 前运行本次改动相关的检查命令。

## AI 开发

AI 辅助开发必须使用 Trellis。Trellis 会把任务计划、研究资料、项目规范、AI 会话记录和检查上下文保存到 `.trellis/` 下的文件中，避免后续会话依赖聊天记录续接。

Trellis 文档：[https://docs.trytrellis.app/](https://docs.trytrellis.app/)

AI 代理修改代码前，应先完成以下步骤：

1. 加载当前 Trellis 上下文：
   ```bash
   python3 ./.trellis/scripts/get_context.py
   python3 ./.trellis/scripts/get_context.py --mode phase
   python3 ./.trellis/scripts/get_context.py --mode packages
   ```
2. 非平凡任务需要在 `.trellis/tasks/` 下创建或继续一个任务。轻量文档改动可以保持轻量，但 AI 代理仍必须读取相关 Trellis 上下文和项目规则。
3. 编辑前阅读适用的 `.trellis/spec/` 索引文件。
4. 及时维护任务产物、实现记录和验证结果。
5. 结束前运行受影响层所需的质量检查。

## Python 环境

Trellis 脚本通过 Python 运行。使用 Trellis 工作流前，请先配置本地虚拟环境。

macOS：

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
```

Windows PowerShell：

```powershell
py -3 -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install --upgrade pip
```

当前仓库没有 Python 依赖清单。如果未来 Trellis 脚本或 hook 提示缺少某个包，请在已激活的虚拟环境中安装该包，并同步记录新增依赖。

## 首次 AI 贡献者设置

每位 AI 贡献者都需要一个个人 Trellis 开发者身份。配置 Python 后，运行一次初始化命令：

```bash
python3 ./.trellis/scripts/init_developer.py <your-name>
```

该命令会创建 `.trellis/.developer` 和 `.trellis/workspace/<your-name>/`。身份文件只保存在本地；workspace 会保存 AI 会话日志和任务记录，帮助未来工作从正确上下文继续。
