# TODO: Windows Portable 版本

## 目标

提供 Windows 免安装压缩包，支持便携模式下把数据写入可执行文件同级 `data/` 目录。

## 范围

- Portable 仅提供 Windows 版本。
- 增加 `--portable` 运行模式。
- 数据目录切到可执行文件同级 `data/`，不写系统 app data 目录。
- 便携版与安装版互不影响。
- 发布清单补充 portable 包产物与 SHA256。

## Rust

- 抽离统一路径解析，所有 DB、settings、resources、state 都通过同一套 path service。
- `--portable` 早于 settings/db 初始化解析。
- autostart 在 portable 模式下默认禁用或提示不可用，避免写系统启动项。

## 打包

- Windows release workflow 增加 zip 产物。
- zip 内包含 exe、必要资源、license/readme。
- 首次运行自动创建 `data/`。

## 验收

- 解压到任意可写目录后可直接运行。
- 移动整个目录后历史、设置、图片资源仍可用。
- 与安装版同时存在时数据目录互不污染。
- `--portable` 缺失时仍使用安装版默认目录。

