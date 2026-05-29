# GitHub Copilot 指令

本项目所有编码约定的单一真相源是仓库根目录的 **[AGENTS.md](../AGENTS.md)**，请完整阅读并遵循其中的：技术栈、仅 macOS+Windows 平台限制、Rust-First 架构原则、目录结构与前后端约定。

分阶段重构清单见 **[TODO.md](../TODO.md)**。

要点速记（详见 AGENTS.md）：
- 核心逻辑尽量下沉 Rust，前端只做 UI；前端禁止直连 SQL、禁止做内容识别与窗口坐标计算。
- 仅支持 macOS + Windows，平台特化用 `#[cfg(target_os = ...)]` 隔离，不要新增 Linux 代码。
- 技术栈：Tauri v2 + React 19 + HeroUI v3 + TailwindCSS v4 + sqlx(SQLite) + Valtio。
