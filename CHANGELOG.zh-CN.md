# 更新日志

## [1.1.0](https://github.com/EcoPasteHub/EcoPaste/compare/v1.0.0...v1.1.0) (2026-07-22)

### ✨ 新功能

* 新增保存图片右键菜单 ([#1353](https://github.com/EcoPasteHub/EcoPaste/issues/1353)) ([746cc05](https://github.com/EcoPasteHub/EcoPaste/commit/746cc0558d656ddb694c47c81a46549b89139c79))
* 新增每日构建更新通道支持 ([#1350](https://github.com/EcoPasteHub/EcoPaste/issues/1350)) ([9544725](https://github.com/EcoPasteHub/EcoPaste/commit/95447258f9e443d5285d46b775c7279d3d14fddc))

### 🐛 问题修复

* 修正旧版数据导入的时区偏移 ([#1338](https://github.com/EcoPasteHub/EcoPaste/issues/1338)) ([d6ea9ed](https://github.com/EcoPasteHub/EcoPaste/commit/d6ea9ed4d1ce6af99673a53a3c1ed78068f83d53))
* 去除重复的 Windows 开机启动项 ([#1355](https://github.com/EcoPasteHub/EcoPaste/issues/1355)) ([0d87458](https://github.com/EcoPasteHub/EcoPaste/commit/0d87458b70323da27456cfaa16b51dcde9ea220e))
* 防止应用卡死导致历史记录丢失 ([#1342](https://github.com/EcoPasteHub/EcoPaste/issues/1342)) ([7a75196](https://github.com/EcoPasteHub/EcoPaste/commit/7a75196e90f4a4abfb49d9f84c591d80c735a468))
* 为 Windows 开机启动路径添加引号 ([#1369](https://github.com/EcoPasteHub/EcoPaste/issues/1369)) ([d9d718e](https://github.com/EcoPasteHub/EcoPaste/commit/d9d718e456173e62f112ebd96609c049a331a1e8))
* 截图采集时重试读取 Windows 剪贴板 ([#1367](https://github.com/EcoPasteHub/EcoPaste/issues/1367)) ([e6f7e80](https://github.com/EcoPasteHub/EcoPaste/commit/e6f7e80e83ae18681a976929d710d99119168866))
* 没有来源应用的剪贴板记录显示 Logo ([#1370](https://github.com/EcoPasteHub/EcoPaste/issues/1370)) ([ad9eee0](https://github.com/EcoPasteHub/EcoPaste/commit/ad9eee0f1b7e3abe6546d4b3ac0fc8d4899bd5cb))
* 跳过旧版数据导入中的无效记录 ([#1326](https://github.com/EcoPasteHub/EcoPaste/issues/1326)) ([c0f2d58](https://github.com/EcoPasteHub/EcoPaste/commit/c0f2d587677ab51f9c707f1e50fda6a83ce78a3e))
* 更新测试版偏好设置文案 ([#1329](https://github.com/EcoPasteHub/EcoPaste/issues/1329)) ([185bb25](https://github.com/EcoPasteHub/EcoPaste/commit/185bb2520e8ce90da8246c0c73b1ce87e54403e5))

## [1.0.0](https://github.com/EcoPasteHub/EcoPaste/compare/v0.6.0-beta.3...v1.0.0) (2026-07-03)

这是 EcoPaste 的全新重构版本，应用整体更轻、更快，也更稳定。

### ✨ 新功能

- Windows 打开 EcoPaste 时支持不抢占其他窗口焦点。
- 新增 Windows Win+V 唤起开关，可用 EcoPaste 替代系统剪贴板面板。
- 新增首次启动引导，帮助完成权限、快捷键、忽略应用和旧版数据导入设置。
- 新增来源应用识别与忽略应用，可按应用控制哪些内容不保存到历史记录。
- 新增采集偏好，可控制保存的内容类型、大小限制和优先顺序。
- 新增敏感内容保护，自动跳过私钥、Token、AWS Key、JWT 等高风险内容。
- 新增完整内容预览，支持预览文本、图片和文件记录。
- 新增拖出能力，可将文本、富文本、图片和文件记录拖到外部应用。
- 新增收藏项与置顶项删除保护。
- 新增偏好设置搜索和重置偏好设置。
- 备份导入导出新增加密备份和合并导入支持。
- 新增 Windows 管理员权限启动设置。
- 软件更新改为独立窗口，可检查、下载并安装新版本。
- 更多新功能，等你下载体验与探索。

### 🐛 问题修复

- 修复 Windows 以管理员权限运行时开机自启动失效的问题。

### ⚡️ 性能优化

- 剪贴板监听、搜索、列表展示和内容预览更快更稳定。
- 应用启动、日常使用和后台资源占用进一步优化。
- 图片缩略图、应用图标和文件图标加载更高效。
- 新增轻量模式，窗口隐藏后减少后台刷新与内存占用。

### ⚠️ 升级说明

- 本重构版仅支持 macOS 和 Windows。
- 本版本支持迁移旧版历史数据，可在首次启动引导中完成导入。
