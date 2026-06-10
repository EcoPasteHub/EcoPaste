# TODO: WebDAV 同步

## 目标

通过用户自有 WebDAV 服务器实现异步备份与多设备合并，补足局域网同步无法覆盖的不同时在线或跨网络场景。

## 前置

优先复用局域网同步的数据模型与合并逻辑：`uuid`、`origin_device_id`、`lamport`、`deleted_at`、`sync_devices`。

## 数据与凭据

- 新表 `sync_webdav_state` 记录每个设备的 push/pull 进度。
- WebDAV 凭据用 `keyring` 保存到 macOS Keychain / Windows Credential Manager。
- 密码、token、同步加密密钥不写入 settings JSON。

## 远端结构

```text
/EcoPaste/
  manifest.json
  devices/<device_id>/
    identity.json
    pages/000000.bincode
    blobs/<sha256>.bin
```

## 客户端

- 使用维护活跃的 WebDAV crate，或基于 `reqwest` 封装 PROPFIND / GET / PUT / DELETE / MKCOL。
- 支持超时、重试、ETag 乐观锁。
- 页文件建议 1MB 左右切分。

## 同步流程

- Push：本地变更 debounce 后，从上次 pushed lamport 起打包增量页并上传。
- Pull：启动与周期拉取 `manifest.json`，按对端进度下载页并合并。
- Blob：按需 GET，本地缓存命中则跳过。
- 远端清理通过 compact 完成，不因本地软删立即删除远端页文件。

## 端到端加密

- 默认启用上传内容加密。
- 用户设置同步密码，用 argon2 派生密钥，页文件和 blob 用 AES-GCM。
- 忘记密码无法恢复云端内容，设置页必须强提示。

## 前端

- 同步设置面板中新增 WebDAV 子区块：
  - 服务器 URL、用户名、密码或 token、根目录。
  - 测试连接。
  - 加密设置。
  - 推送/拉取间隔。
  - 立即推送、立即拉取、compact。
- 长错误使用 Modal 展示完整 HTTP 响应摘要。

## 验收

- 两台设备不同时在线也能通过 WebDAV 合并增量。
- 凭据不落明文。
- 加密开启时远端页文件不可直接读取内容。
- 网络失败不影响本地剪贴板主流程。

