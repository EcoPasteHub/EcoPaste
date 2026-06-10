# TODO: 局域网同步

## 目标

同一局域网内的 EcoPaste 设备自动发现、配对，并同步剪贴板新增条目与元数据变更。

## 范围

- 仅 macOS 与 Windows。
- 发现、传输、合并都在 Rust 实现。
- 前端只做配对 UI、状态展示和设置。
- 锁定条目默认不同步，除非后续完成跨设备加密设计。

## 数据模型

- `clipboard_items` 增加：
  - `uuid TEXT UNIQUE NOT NULL`
  - `origin_device_id TEXT NULL`
  - `lamport INTEGER NOT NULL DEFAULT 0`
  - `deleted_at TEXT NULL`
- 新表 `sync_devices` 记录配对设备。
- 当前版本仍是 `0.6.0-beta.3`，按项目约定直接改 `0001_init.sql`。

## 发现与配对

- 使用 mDNS 广播 `_ecopaste._tcp.local`。
- TXT 记录包含 `device_id`、`device_name`、`version`、公钥指纹。
- 首次配对使用 6 位 PIN。
- 配对后保存对端公钥与元数据。

## 传输与协议

- 使用 QUIC 建立设备间连接。
- 每台设备持久化长期身份密钥。
- 协议消息包括 `Hello`、`RequestSince`、`Items`、`Ack`、`MetaUpdate`、`RequestBlob`、`BlobChunk`。
- 内容按 `uuid` 去重，元数据使用 Lamport + LWW 合并。
- 图片 blob 按需拉取，避免默认传大文件。

## 隐私过滤

- 推送前应用敏感类型、排除应用、文件大小等过滤。
- 与每应用规则联动，禁捕获应用不进入同步。

## 前端

- 设置页新增同步面板：
  - 总开关。
  - 本机设备名。
  - 已配对设备。
  - 同网段发现设备。
  - 配对请求与 PIN 弹窗。
- 主窗可显示轻量同步状态图标。

## 平台注意

- macOS 需要本地网络权限描述与 Bonjour service 声明。
- Windows 可能触发防火墙提示，安装器应尽量提供规则或说明。

## 验收

- 两台设备可发现、配对、断线重连并补齐增量。
- 新条目、收藏、置顶、备注、删除状态能合并。
- 图片按需拉取，超限有明确提示。
- 冲突合并幂等，不重复入库。

