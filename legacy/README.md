# Legacy reference

这个目录保留迁移前的剪贴板核心与前端 UI 参考，供 Go/NSGo 与 AppKit 迁移时对照行为。

**它不是可构建工程。** 根目录的 `src/`、`src-tauri/`、Cargo/Tauri 配置、窗口、菜单、托盘和发布脚本都已移出仓库。这里的代码可能引用已删除模块，不要为了让它编译而恢复旧工程。

## 剪贴板核心

| 当前路径 | 原路径 | 用途 |
| --- | --- | --- |
| `legacy/backend/src/clipboard/` | `src-tauri/src/clipboard/` | 监听、读取、写回、类型识别、去重、回环抑制、来源应用、图片/文件存储 |
| `legacy/backend/src/db/` | `src-tauri/src/db/` | SQLite 模型、FTS5 查询、分组、来源应用、文件图标 |
| `legacy/backend/src/settings/` | `src-tauri/src/settings/` | 设置模型、默认值、patch/merge、原子持久化 |
| `legacy/backend/src/backup/` | `src-tauri/src/backup/` | `.ecopastebak` 明文/加密容器与导入导出 |
| `legacy/backend/src/keystroke/` | `src-tauri/src/keystroke/` | macOS 模拟粘贴 |
| `legacy/backend/src/core/` | `src-tauri/src/core/` | 数据路径与错误类型；已裁剪为最小引用 |
| `legacy/backend/src/commands/clipboard.rs` | `src-tauri/src/commands/clipboard.rs` | 列表补字段、可用动作、条目操作等命令层行为 |
| `legacy/backend/migrations/0001_init.sql` | `src-tauri/migrations/0001_init.sql` | 已发布 SQLite schema、FTS5 与触发器 |

## 前端 UI

| 当前路径 | 原路径 | 用途 |
| --- | --- | --- |
| `legacy/frontend/src/pages/` | `src/pages/` | Clipboard、Preview、Preference、Onboarding、Update 页面参考 |
| `legacy/frontend/src/components/` | `src/components/` | 卡片、列表、快捷键、分组图标、弹窗等视觉参考 |
| `legacy/frontend/src/hooks/`、`stores/` | `src/hooks/`、`src/stores/` | 分页、键盘、窗口生命周期、状态划分参考 |
| `legacy/frontend/src/commands/`、`constants/`、`types/` | `src/commands/`、`constants/`、`types/` | 命令、事件与 DTO 契约参考 |
| `legacy/frontend/src/locales/` | `src/locales/` | zh-CN / en-US 文案来源 |
| `legacy/frontend/src/styles/`、`unocss/` | `src/styles/`、`src/unocss/` | 颜色、字号、间距、主题 token 参考 |
| `legacy/frontend/public/` | `public/` | 旧版 Logo、图标与插画资源 |
| `legacy/frontend/*.config.*`、`tsconfig*` | 根目录配置 | 依赖、别名、样式与 TS 边界参考 |

前端不接入构建，不执行 `pnpm install` / `pnpm dev`，也不要把 `legacy/frontend/src` 移回仓库根目录。

## 已移出的内容

- Tauri 外壳、窗口、菜单、托盘、快捷键、自启、更新、i18n、拖出和平台专用代码。
- `src-tauri/Cargo.toml`、`tauri.conf.json`、icons、assets、capabilities。
- GitHub workflows、release 配置、旧项目文档、AI 工具目录和 `.trellis`。

完整旧工程归档在仓库外，路径见 `docs/nsgo-migration/06-cleanup-report.md`。

## 使用方式

- 迁移剪贴板行为时，先读 `legacy/backend/`，再对照 `docs/nsgo-migration/01-current-architecture.md` 与 `04-feature-parity-checklist.md`。
- 迁移 UI 时，先读 `legacy/frontend/src/pages` 与 `components`，再按 `docs/nsgo-migration/04-feature-parity-checklist.md` 的 U-01 ~ U-12 验收。
- 不直接复制 Rust/React 代码；先提取数据契约、状态和交互，再用 Go/AppKit 原生方式实现。
- 不修改本目录来“修好”旧工程；它只作为历史参考。
