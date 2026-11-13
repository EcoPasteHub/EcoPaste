# 06 · 清理报告

> 日期：2026-11-13
> 分支：`refactor/nsgo-migration`
> 目标：仓库保留迁移文档、`legacy/` 剪贴板核心与前端 UI 参考、一份 `AGENTS.md`，不保留可运行的旧工程。

## 1. 最终保留结构

```text
EcoPaste/
  AGENTS.md
  README.md
  LICENSE
  .gitattributes
  .gitignore
  docs/nsgo-migration/
  legacy/
    README.md
    backend/
      src/clipboard/
      src/db/
      src/settings/
      src/backup/
      src/keystroke/
      src/core/{error.rs,paths.rs,mod.rs}
      src/commands/clipboard.rs
      migrations/0001_init.sql
    frontend/
      README.md
      src/
      public/
      index.html
      package.json
      uno.config.ts
      vite.config.ts
      tsconfig.json
      tsconfig.node.json
      biome.json
```

## 2. 剪贴板核心参考

`legacy/backend/` 只保留与剪贴板行为、数据契约、设置和备份直接相关的代码：

| 模块 | 保留原因 |
| --- | --- |
| `clipboard/` | 监听、读取/写回、类型识别、去重、回环抑制、来源应用、图片/文件存储、清理、敏感内容 |
| `db/` | SQLite 模型、FTS5 查询、分组、来源应用、文件图标 |
| `settings/` | 设置模型、默认值、patch/merge、原子持久化 |
| `backup/` | `.ecopastebak` 明文/加密容器与导入导出规则 |
| `keystroke/` | macOS 模拟粘贴 |
| `core/{error,paths}` | 错误类型与数据目录解析 |
| `commands/clipboard.rs` | 命令层补字段、可用动作、条目操作行为 |
| `migrations/0001_init.sql` | 已发布 SQLite schema、FTS5 和触发器 |

这部分不参与构建，不要求编译通过；它引用的窗口、菜单、i18n、托盘等模块已移出。

## 3. 前端 UI 参考

`legacy/frontend/` 恢复旧 React 前端，用于对照布局、组件、交互状态、文案和视觉细节。

| 内容 | 用途 |
| --- | --- |
| `src/pages/Clipboard`、`Preview`、`Preference`、`Onboarding`、`Update` | 页面结构与交互流程参考 |
| `src/components/` | 卡片、列表、快捷键录制、分组图标、弹窗等视觉参考 |
| `src/hooks/` | 分页缓存、键盘导航、窗口生命周期、主题等行为参考 |
| `src/stores/` | UI 状态划分参考 |
| `src/commands/`、`src/constants/`、`src/types/` | 命令、事件与 DTO 契约参考 |
| `src/locales/`、`src/i18n/` | zh-CN / en-US 文案来源 |
| `src/styles/`、`src/unocss/` | 颜色、字号、间距、主题 token 参考 |
| `public/` | 旧版图标、Logo 与插画资源 |
| `package.json`、`vite.config.ts`、`uno.config.ts`、`tsconfig*.json`、`biome.json` | 依赖、别名、构建与样式配置参考 |

前端参考同样不接入构建：

- 不执行 `pnpm install` / `pnpm dev`。
- 不把 `legacy/frontend/src` 移回根目录。
- 不恢复 Tauri 命令、事件和 WebView API；UI 实现改用 NSGo + AppKit。
- 旧前端只有参考价值，不作为“第二套可运行应用”。

## 4. 已移出的内容

完整旧工程归档于：

```text
/tmp/ecopaste-legacy-20261113-123221
```

确认无需回退后可手动删除该归档目录。

### 4.1 AI 工具与协作目录

```text
.trellis/
.agents/
.claude/
.codex/
.cursor/
.gemini/
.kiro/
.opencode/
.github/         # 含 Trellis agents/skills 与旧 CI workflows
.vscode/
```

仓库内不再保留 Trellis 任务、规范、hooks、skills 或其它 AI 工具入口。AI 规则只保留根目录 `AGENTS.md`。

### 4.2 旧 Tauri 外壳与平台代码

从旧 `src-tauri/` 中移出的部分包括：

```text
Cargo.toml / Cargo.lock / build.rs
tauri.conf.json / capabilities / icons / assets
lib.rs / main.rs
window / menu / tray / shortcut / autostart / update
i18n / drag_out / keyboard / mouse / admin.rs
commands 的其余文件
clipboard/db/settings/backup/keystroke 之外的模块
```

### 4.3 构建、发布与旧文档

```text
scripts/
pnpm-lock.yaml
pnpm-workspace.yaml
rust-toolchain.toml
.release-it.ts
lint-staged.config.ts
simple-git-hooks.json
.commitlintrc
CHANGELOG.md
CHANGELOG.zh-CN.md
CONTRIBUTING.md
CONTRIBUTING.zh-CN.md
README.zh-CN.md
```

`README.md` 已重写为当前迁移状态；`LICENSE` 保留。

## 5. 保留的根文件

| 文件 | 原因 |
| --- | --- |
| `AGENTS.md` | 唯一 AI 开发规范 |
| `README.md` | 当前仓库说明与迁移入口 |
| `LICENSE` | Apache-2.0 许可 |
| `.gitignore` | 忽略构建产物与本地文件 |
| `.gitattributes` | 统一文本换行与二进制标记 |
| `docs/nsgo-migration/` | 迁移分析、计划、清单、风险和本报告 |
| `legacy/backend/` | 剪贴板与数据契约参考 |
| `legacy/frontend/` | UI 与交互参考 |

## 6. 为什么没有保留可运行的旧应用

- 旧版完整源码、构建配置和发布链路已归档，需要时可以从 `/tmp` 或 git history 恢复。
- 保留半套 Tauri 工程会制造“仓库里有两套架构”的歧义，也会让 CI、依赖和构建命令重新膨胀。
- 前端只作为静态参考，不能直接运行；新 UI 必须用 NSGo + AppKit 重写。
- 后续全部迁移工作只在 `refactor/nsgo-migration` 分支推进，不再为每个 Phase 开分支。

## 7. 后续删除 `legacy/` 的条件

- `04-feature-parity-checklist.md` 中 P0/P1 项全部通过。
- Go/NSGo 实现已覆盖剪贴板采集、写回、去重、回环、搜索、分组、收藏/置顶/备注、备份。
- AppKit UI 已覆盖前端参考中的页面、状态和交互。
- 数据兼容与备份互操作测试通过。
- 迁移文档不再需要引用对应旧实现。

满足后再整体删除 `legacy/`，不要逐文件零散删除。

## 8. 验证

清理后检查：

- 仓库根目录不存在 `.trellis/`、`.agents/`、`.claude/`、`.codex/`、`.cursor/`、`.gemini/`、`.kiro/`、`.opencode/`。
- 仓库根目录不存在 `src/`、`public/`、`package.json`、`src-tauri/`。
- `legacy/backend/` 只包含剪贴板核心参考文件。
- `legacy/frontend/` 只包含 UI 参考文件，且没有 `node_modules`、`dist`。
- `AGENTS.md` 是唯一 AI 指令文件，且不再引用 Trellis。
- `docs/nsgo-migration/` 保留完整迁移计划与路径说明。
