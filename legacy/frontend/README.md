# Legacy frontend UI reference

这个目录保留迁移前的 React 前端，作为 AppKit UI 迁移的静态参考。

**它不是可运行应用。** 没有 root `package.json`、lockfile、node_modules 或 Tauri 外壳；不要在这里执行 `pnpm install` / `pnpm dev`，也不要把 `src/` 移回仓库根目录。

## 参考什么

- `src/pages/Clipboard/`：主面板布局、搜索、分类、分组、列表、卡片、Footer。
- `src/pages/Preview/`：预览内容、定位、hover/Space 状态。
- `src/pages/Preference/`：设置分组、搜索、高亮、存储/备份/权限/更新界面。
- `src/pages/Onboarding/`：首次启动与权限流程。
- `src/components/`：按钮、卡片、快捷键录制、分组图标等视觉与交互。
- `src/hooks/`：分页缓存、键盘导航、窗口生命周期、主题等行为。
- `src/stores/`：UI 状态划分。
- `src/locales/`：zh-CN / en-US 文案来源。
- `src/styles/`、`src/unocss/`：颜色、字号、间距和主题 token。
- `public/`：旧版 Logo、图标和插画资源。

`package.json`、`vite.config.ts`、`uno.config.ts`、`tsconfig*.json`、`biome.json` 只用于理解依赖、路径别名、样式配置和 TS 边界，不参与新构建。

## 迁移方式

| 旧前端 | 新实现 |
| --- | --- |
| React 组件 | NSGo `core` 结构 + AppKit 控件 |
| Valtio store | NSGo signals / Go 状态 |
| Ant Design / UnoCSS | AppKit 语义色、系统字体与原生布局 |
| `react-virtuoso` | `NSTableView` / `NSCollectionView` |
| WebView 预览 | `NSImageView` / `NSTextView` |
| Tauri command/event | Go service + NSGo 事件编排 |

只参考视觉层级、文案、状态和交互顺序，不直接翻译组件树或复制 CSS。
