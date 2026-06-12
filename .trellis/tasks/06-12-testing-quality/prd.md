# Testing and Quality Assurance

## Goal

补齐发布前质量保障：单元测试、命令集成测试、前端组件测试、跨平台手动验收和性能基线。

## Scope

- 持续补齐 Rust 仓储层与内容识别单元测试。
- 覆盖关键 Tauri IPC 行为的命令集成测试。
- 使用 Vitest + Testing Library 建立前端关键组件测试基础设施。
- 固化 macOS / Windows 手动验收清单。
- 建立大量历史记录下列表滚动与搜索延迟测试。
- 建立内存 / CPU 占用基线，可与旧版或本仓库后续版本对比。

## Implementation Notes

- 先固化现有主路径手动验收清单。
- 优先补 Rust 高风险模块测试：数据库、剪贴板识别、设置合并、窗口几何。
- 再建立前端组件测试基础设施。
- 加入命令层集成测试或可重复 smoke test。
- 建立性能脚本与数据集。

## Acceptance Criteria

- `pnpm lint`、`pnpm tsc`、`cargo fmt --check`、`cargo clippy -- -D warnings`、`cargo test` 成为发布前固定检查。
- 手动验收覆盖 macOS 与 Windows。
- 性能基线记录在文档或 CI artifact 中。
- 新增高风险功能时有明确测试增量。
