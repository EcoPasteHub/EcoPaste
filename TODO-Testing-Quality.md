# TODO: 测试与质量保障

## 目标

补齐发布前质量保障：单元测试、命令集成测试、前端组件测试、跨平台手动验收和性能基线。

## 范围

- Rust 仓储层与内容识别单元测试持续补齐。
- Tauri 命令集成测试覆盖关键 IPC 行为。
- 前端关键组件测试使用 Vitest + Testing Library。
- macOS / Windows 手动验收清单。
- 大量历史记录下列表滚动与搜索延迟测试。
- 内存/CPU 占用对比旧版或建立本仓库基线。

## 建议顺序

1. 固化手动验收清单，先覆盖现有主路径。
2. 给 Rust 高风险模块补测试：数据库、剪贴板识别、设置合并、窗口几何。
3. 建立前端组件测试基础设施。
4. 加入命令层集成测试或可重复的 smoke test。
5. 建立性能脚本与数据集。

## 验收

- `pnpm lint`、`pnpm tsc`、`cargo fmt --check`、`cargo clippy -- -D warnings`、`cargo test` 成为发布前固定检查。
- 手动验收覆盖 macOS 与 Windows。
- 性能基线记录在文档或 CI artifact 中。
- 新增高风险功能时有明确测试增量。

