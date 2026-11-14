# EcoPaste

macOS 优先的本地剪贴板管理器。

项目正在从 **Tauri v2 + Rust + React** 迁移到 **Go + NSGo + AppKit**。新的 Go/NSGo 应用已建立菜单栏应用壳，正在按垂直切片补齐剪贴板能力。

## 当前仓库

| 路径 | 内容 |
| --- | --- |
| `cmd/ecopaste/` | NSGo 应用入口与打包配置 |
| `internal/app/` | 应用编排、托盘与快捷键 |
| `internal/native/macos/` | AppKit 原生视图桥接 |
| `internal/storage/` | 数据目录与 storage 锚点 |
| `internal/settings/` | `settings.json` 模型、patch 与持久化 |
| `docs/nsgo-migration/` | 迁移分析、目标架构、阶段计划、功能对照、风险与清理记录 |
| `legacy/backend/` | 旧剪贴板核心参考代码，不参与构建 |
| `legacy/frontend/` | 旧 React UI 参考，不参与构建 |
| `AGENTS.md` | 项目长期开发规范 |

旧版完整工程与 Tauri 外壳已移出仓库；剪贴板核心和前端 UI 以 `legacy/` 参考保留，归档位置见 `docs/nsgo-migration/06-cleanup-report.md`。

## 目标

- 当前阶段只支持 macOS。
- UI 使用 AppKit 原生控件与系统能力。
- NSGo 负责应用结构、状态、布局和事件编排。
- 保留剪贴板、历史、搜索、分组、收藏、置顶、备注、图片/文件、备份等核心能力。
- 开发阶段优先新架构与核心体验，旧版数据兼容在合并主分支、正式发布前再补齐。

## 开发

需要 Go 1.26+ 与 NSGo CLI：

```bash
go install github.com/nsgo-dev/nsgo/cmd/nsgo@70cef6a4ed17f7c8428faabc9bb419ff4223745b
nsgo check cmd/ecopaste
nsgo dev -id com.ayangweb.eco-paste.dev cmd/ecopaste
nsgo build -id com.ayangweb.eco-paste.dev -output build/dev cmd/ecopaste
```

也可以优先使用 `make`：`make dev`、`make build-dev`、`make check`，`make help` 查看全部。`make build` 只用于正式身份构建。

本地开发构建使用独立 bundle ID `com.ayangweb.eco-paste.dev`，数据写入 `~/Library/Application Support/com.ayangweb.eco-paste/dev`。开发身份优先于 `ECOPASTE_MODE`，即使设置 `ECOPASTE_MODE=prod` 也不会读取正式数据。开发构建如果误用正式 bundle ID 会直接退出，避免与已安装版本共享 LaunchServices、登录项或设置。`ECOPASTE_HOME` 可指向测试用目录。

迁移优先级与验收项见 `docs/nsgo-migration/03-migration-plan.md` 与 `04-feature-parity-checklist.md`。

## License

[Apache License 2.0](./LICENSE)
