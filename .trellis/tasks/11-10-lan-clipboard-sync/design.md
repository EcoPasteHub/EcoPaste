# Design: LAN 剪贴板同步

## 模块与边界（Rust-First）

新增 `src-tauri/src/sync/`，前端只做设置页与弹窗。

```text
sync/
  mod.rs        # init(app)、SyncState（Tauri State）、对外 API（broadcast_item）
  identity.rs   # 本机 device_id / 设备名 / 长期密钥；devices.json 持久化
  discovery.rs  # mdns-sd 注册与浏览 `_ecopaste-sync._tcp`
  pairing.rs    # 配对码生命周期（生成/过期/尝试计数）+ SPAKE2 配对握手
  session.rs    # 已配对设备间的认证握手 + AEAD 帧通道
  protocol.rs   # 消息定义（serde + 长度前缀帧）
  server.rs     # TCP 监听、接入分发（配对 / 会话）
  peers.rs      # 连接管理、重连退避、广播
  apply.rs      # 收到条目 → 入库 + 写剪贴板（复用 clipboard::write / guard / ingest）
  files/
    mod.rs      # 文件清单、按需拉取（FetchFile）
    windows.rs  # IDataObject 虚拟文件（FileGroupDescriptorW + FileContents IStream）
    macos.rs    # NSFilePromise —— 先 TODO
commands/sync.rs  # 薄层 command
i18n/…            # 托盘/toast 文案（如有）
```

依赖（版本写主版本号）：`mdns-sd`、`spake2`、`hkdf`、`sha2`、`hmac`、`tokio`（补 `net`/`io-util`/`sync`/`rt-multi-thread`/`fs`）。已有 `chacha20poly1305`、`rand`、`zeroize`、`blake3`。

## 设备身份与持久化

- `device_id`：UUID，首次启动生成；`device_name` 默认取主机名，可改。
- 配对设备表 `sync/devices.json`（app 数据目录，原子写）：`{device_id, name, platform, key(32B), paired_at}`。`key` 即配对时协商出的长期共享密钥。明文存盘，首版接受；后续可换系统钥匙串（记 follow-up）。
- 设置镜像（`settings.sync`）只存：`enabled: bool`、`device_name: String`、`sync_text/image/files: bool`。新字段均 `#[serde(default)]`，无需 migration。无 DB schema 变更。

## 发现

- 开启同步后：TCP 监听 `0.0.0.0:0`，端口用 mDNS 通告。
- 服务名 `_ecopaste-sync._tcp.local.`，实例名 = `device_id` 短哈希；TXT：`id`（device_id）、`name`、`pair`（`1` 表示正在展示配对码）、`v`（协议版本）。
- 浏览结果维护 `id → {addrs, port, pair}`；已配对设备出现 / 地址变化即触发连接。
- Windows 首次监听会触发防火墙放行提示；文档/引导需说明。

## 配对（6 位码 → SPAKE2 → 长期密钥）

1. A「查看匹配码」：`pairing::start()` 生成 6 位 CSPRNG 数字（含前导零），`expires_at = now+5min`，`attempts = 0`；mDNS TXT 置 `pair=1`。
2. B 输入码：对所有 `pair=1` 的发现设备并发尝试连接，发 `PairHello{device_id,name,platform}` 后做 **SPAKE2**（口令 = 6 位码，双方身份串 = device_id）。
3. 双方得到会话密钥 `K`；互发 `HMAC(K, transcript)` 作 key confirmation。验证失败 → A 的 `attempts += 1`，≥3 次或过期则码作废（撤 `pair=1`），连接关闭。
4. 成功后双方各生成 16B 随机数并交换，`long_key = HKDF(K, salt=nonceA||nonceB, info="ecopaste-sync-v1-longterm")`，用 `K` 派生的 AEAD 密钥加密传输设备名等元信息，各自写入 `devices.json`。码立即失效。
5. PAKE 性质：被动窃听者无法从抓包离线爆破 6 位码；主动在线猜测受 3 次上限与 5 分钟窗口约束。

## 会话（已配对设备间）

- 连接：`Hello{device_id, nonce_c}` → `Challenge{nonce_s}`；双方用 `long_key` 做 `HMAC(long_key, "c"||nonce_c||nonce_s)` / `"s"…` 双向认证（防重放、防冒充）。
- 会话密钥 `HKDF(long_key, nonce_c||nonce_s, "session")`，分方向两把子密钥；帧 = `u32 len` + `ChaCha20-Poly1305(nonce=64bit 递增计数 + 方向前缀)`。单帧上限 8 MiB，大负载分块。
- 心跳 15s；对端无响应 45s 断开重连，指数退避（1s→30s）。
- 去重连：两端互相发现时，按 `device_id` 字典序较小者作发起方，避免双连接。

## 协议消息（postcard 或 serde_json 帧体，首版 JSON 便于调试）

- `ClipText{hash, plain, html?, rtf?}`
- `ClipImage{hash, width, height, size}` + 随后 `Chunk{id, seq, bytes}`…`ChunkEnd`
- `ClipFiles{hash, offer_id, entries:[{index, rel_path, is_dir, size}]}`（仅清单）
- `FetchFile{offer_id, index, offset, len}` → `FileChunk{…}`/`FileEnd`；`CancelOffer{offer_id}`
- `Ping/Pong`、`Unpair`

## 发送侧挂钩

在 `clipboard::watcher` 的 `persist_and_notify` 成功后（item 非 guard 跳过）调用 `sync::broadcast_item(app, &item)`：

- 过滤：`enabled`、类型开关、`is_sensitive`、大小上限（沿用 `capture`）。
- 文本：直接取 `content`/`search_text`/`sub_kind` 映射到 `ClipText`；图片：读原图 PNG 流式发；文件：生成清单（递归展开目录），登记 `offer_id → 本地路径`（保留到下次复制或 10 分钟无访问），**不读文件内容**。
- 单进程 `SendQueue`：新复制会取消上一个未完成的图片传输；文件 offer 保留最近 N=3 个。

## 接收侧（apply.rs）

- 文本 / 图片：构造 `ClipboardItem`（`platform` 取对端平台、来源 app 为空），`guard.suppress(content_hash)` → 写系统剪贴板（`write_to_clipboard`）→ `persist_and_notify`。因被 guard 抑制，不会被 watcher 再入库，也就不会再广播（星型无转发，天然无环）。
- 图片落盘走 `ImageStore::store`，复用 `content_hash` 与监听回环抑制逻辑。
- 文件：不入历史；在剪贴板上放虚拟文件对象（见下），并 `guard.suppress` 对应哈希（虚拟文件对象本身对 watcher 不可读为路径，需验证 `clipboard-rs` 读取不会误入库，必要时在 watcher 里用自定义 format 标记跳过）。

## 虚拟文件（Windows）

- 自实现 `IDataObject`（`windows` crate：`Win32_System_Ole`、`Win32_System_Com`、`Win32_UI_Shell` 已启用）：
  - `CFSTR_FILEDESCRIPTORW`：每个条目一个 `FILEDESCRIPTORW`（相对路径、大小、`FILE_ATTRIBUTE_DIRECTORY`）。
  - `CFSTR_FILECONTENTS`（`lindex`）：返回 `IStream`，`Read` 时按需通过会话发 `FetchFile` 拉块（带小缓冲、顺序读；Seek 仅支持 0 起点/向前）。
  - `CFSTR_PERFORMEDDROPEFFECT` / 取消时发 `CancelOffer`。
- 用 `OleSetClipboard` 放置；需要在持有 STA 的专用线程上创建、泵消息。`clipboard-rs` watcher 线程与此独立。
- 失败回退：流读取失败返回 `STG_E_READFAULT`；对端离线时 Ctrl+V 报错，不崩溃。
- 本期风险最大的部分：先 PoC（Explorer 粘贴 / 聊天软件粘贴 / 大文件取消），再接协议。

## 虚拟文件（macOS）

- 方案：`NSFilePromiseProvider` + `NSPasteboard writeObjects`，delegate 在 `writePromiseToURL` 时拉取。当前环境无 Mac 验证，**首版显式 TODO**，macOS 文件同步降级为：接收端提示「该设备文件需在 Windows 上粘贴」不处理（或直接不发文件给 macOS）。待有 macOS 环境再补，需在 PRD 中保持可见。

## Commands / Events（跨端契约）

常量集中：`src-tauri/src/sync/mod.rs` + `src/constants/{commands,events}.ts`。

- Commands：`sync_get_status`、`sync_set_enabled`（走 `update_settings` 亦可）、`sync_start_pairing`→`{code, expiresAt}`、`sync_cancel_pairing`、`sync_pair_with_code(code)`、`sync_rename_device`、`sync_rename_peer(id,name)`、`sync_remove_peer(id)`。
- Events：`sync://devices`（设备列表/在线状态变化）、`sync://pairing`（`{state: waiting|success|expired|failed, peer?}`）。

## 前端

- Preference 新增 tab `sync`（`src/pages/Preference/types` + `PREFERENCE_TAB_META` + `config/preferenceSchema.ts` + 侧栏），组件 `SyncPanel`、`PairingCodeModal`（6 格展示 + 倒计时）、`PairCodeInputModal`（6 位输入）、`DeviceList`。
- 遵循 AGENTS.md 前端规范（antd v6、UnoCSS token、`cn`、Valtio 仅镜像设置、`@/utils/log`）；`zh-CN` + `en-US` 文案同补。

## 兼容与回滚

- 设置新增 `sync` 段默认关闭，旧配置照常读取；无 DB migration。
- 协议带版本号 `v`，不兼容则拒绝并提示。
- 关闭开关即停止监听 / mDNS / 断开所有连接；`devices.json` 保留。

## 关键取舍

- PAKE 而非「码当预共享密钥」：6 位码熵太低，直接做密钥会被抓包后离线爆破。
- 自研轻量协议而非 TLS：无 CA，PAKE 产物天然做双向认证；依赖少。风险：自研密码学协议需评审——只用成熟原语（SPAKE2、HKDF、HMAC、ChaCha20-Poly1305），不自造。
- 文件走虚拟文件而非预传：满足「按 Ctrl+V 才传」，代价是平台底层工作量大。
