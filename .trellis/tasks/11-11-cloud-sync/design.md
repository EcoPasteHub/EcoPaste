# Design: 云同步

## 仓库与布局

- 新仓库 `D:\Projects\claude\EcoPasteServer`（远端 `ack528/EcoPasteServer`），不放进 EcoPaste 仓库。结构镜像 MediaHub：

```text
EcoPasteServer/
  server/            # Go：cmd/ecopaste-server, internal/{api,auth,store,blob,hub,retention,tlsx,update,config,logx}
  desktop/           # Tauri 2 管理程序（React + antd），通过本机 admin 通道控制服务
  tools/ docs/ dist/ # 便携工具链、文档、打包产物（同 MediaHub）
```
- 可直接复用 MediaHub 的 `tlsx`（自签证书 + 指纹）、`update`（GitHub Releases 自动更新 + 回退）、`logx`、`auth` 的思路与代码风格，不跨仓库引用，拷贝后裁剪。

## 协议（HTTPS + WebSocket，JSON；二进制走 blob）

| 接口 | 说明 |
| --- | --- |
| `POST /api/login` | `{username,password,deviceName,platform}` → `{token,deviceId}`；失败限流 |
| `GET /api/ws?since=<seq>` | 令牌鉴权；先补发 `seq>since` 的条目，再实时推送；心跳 |
| `POST /api/items` | 上传一条：`{hash,kind,subKind,text{plain,html,rtf},image{blob},files{entries}}` → `{seq}`；按 `hash` 去重 |
| `PUT /api/blobs/{sha256}` / `GET` | 图片 / 文件内容，支持 `Range`；校验哈希与单文件上限 |
| `GET /api/history?limit=&before=` | 新设备拉取历史 |
| `DELETE /api/devices/{id}`（管理端） | 吊销令牌，WebSocket 立即断开 |

- 条目有全局递增 `seq`（每账号），客户端保存 `last_seq`，断线重连带 `since`。
- 文件条目：只上传清单；内容仅在源设备在线时可经服务端中转按需拉取（`FetchFile` 经 WS 转发，服务端不落盘），或在容量限额内预传 blob（管理端开关，默认关闭）。首期实现中转，不落盘。

## 安全

- 仅 HTTPS：自签证书 → 客户端首次连接展示指纹并确认，之后固定；自带证书则走系统信任链。
- 口令 argon2id；令牌随机 256 位，服务端只存哈希；每设备一个令牌，可吊销。
- 登录限流：按 IP + 用户名双维度，递增锁定；`MaxBytesReader`、每账号速率限制、WS 消息大小上限；审计日志（登录、吊销、清理）。
- 明文策略：内容对服务端可见（已确认）。客户端对 `is_sensitive` 不上传；管理端不提供内容浏览，仅统计。

## 客户端（EcoPaste，新模块 `src-tauri/src/cloud/`）

```text
cloud/
  client.rs    # reqwest(rustls) + tokio-tungstenite，指纹固定 verifier
  session.rs   # 登录、令牌持久化（设备表同目录 state/cloud.json）、重连退避、since 追赶
  sync.rs      # 本机新条目 → POST；WS 收到 → apply（复用 sync::apply 的写剪贴板 + 入库 + guard）
  mod.rs       # init / 设置变更 / 状态
commands/cloud.rs
```

- 复用：`sync::apply`（文本 / 图片写入）、`remote_files`（虚拟文件，`fetch_file_range` 抽象成 trait：LAN 会话 / 云端中转两种实现）。
- 去重 / 防回环：条目携带来源 `origin`（device_id）与 `content_hash`；LAN 与云端同时到达时，以 `content_hash` + 短 TTL 去重；`WritebackGuard` 照旧。
- 设置：`settings.cloud { enabled, serverUrl, certFingerprint, username, syncText/Image/Files, pullRecent }`；令牌 / 口令不进 settings。
- 前端：「跨设备」页新增「云同步」分组（登录表单、指纹确认弹窗、状态、拉取历史按钮），遵循现有 UI 规范与 i18n。

## 管理程序

- Tauri 2 + React/antd，与 MediaHub `desktop` 同构：本机回环 admin API（仅 127.0.0.1 + 管理令牌）控制账号、设备、保留策略、日志、更新、证书；托盘常驻。
- 打包同 MediaHub：`tools\package.ps1` → 便携 zip + 更新 zip + `SHA256SUMS.txt`；`WebView2Loader.dll` 必须带上（GNU 构建教训）。

## 兼容 / 回滚

- 客户端默认关闭云同步，不影响局域网同步；`settings.cloud` 缺省回落。
- 服务端 DB 带版本迁移；更新失败自动回退（参照 MediaHub）。

## 风险

- 公网暴露 + 服务器可见明文：已由用户选择；通过建号制、限流、吊销、不上传敏感内容缓解，并在管理端与客户端首次登录时明确提示。
- 大图 / 大文件带宽与存储：限额 + LRU + 文件中转不落盘。
- 与局域网同步并存的去重 / 回环：用 e2e 测试覆盖。
