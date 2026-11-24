# Implementation Plan

按可独立验证的里程碑推进，每步先 `cargo clippy -- -D warnings && cargo test`。

## M1 基础：身份 / 协议 / 加密通道（纯 Rust，可单测）
- [ ] 依赖：`mdns-sd`、`spake2`、`hkdf`、`sha2`、`hmac`，tokio 加 `net`/`io-util`/`sync`/`rt-multi-thread`/`fs`/`macros`
- [ ] `settings.sync` 模型 + 默认值 + 前后端字段同步
- [ ] `identity.rs`、`protocol.rs`、`session.rs`（帧加解密、重放计数）
- [ ] 单测：帧往返、篡改拒绝、错误密钥拒绝、认证握手

## M2 配对
- [ ] `pairing.rs`：码生命周期、3 次作废、SPAKE2 + key confirmation、写 `devices.json`
- [ ] 单测：双端内存管道配对成功/错误码/过期/超次数
- [ ] `discovery.rs`：mDNS 注册/浏览，`pair` TXT 切换

## M3 连接与文本/图片同步
- [ ] `server.rs`、`peers.rs`（发起方仲裁、重连退避、心跳）
- [ ] 发送钩子接入 `watcher`；`apply.rs` 接收写剪贴板 + 入库 + guard
- [ ] 过滤：敏感、类型开关、大小上限
- [ ] 本机双进程 / 双实例集成测试（回环地址 + 手动 mDNS 绕过）

## M4 命令与前端
- [ ] `commands/sync.rs` + `generate_handler!` 注册；事件常量两端同步
- [ ] Preference「跨设备」页、配对码弹窗、输入弹窗、设备列表
- [ ] i18n zh-CN / en-US；`pnpm lint`
- [ ] 实际操作验证主路径与边界（改 UI 必做）

## M5 文件虚拟粘贴（Windows）
- [ ] PoC：自实现 IDataObject，Explorer 粘贴一个固定内存文件
- [ ] 接协议 `ClipFiles` / `FetchFile`，目录递归、取消、对端离线报错
- [ ] 大文件（≥1GB）内存占用与速度验证
- [ ] macOS：显式 TODO + 降级行为

## M6 收尾
- [ ] `trellis-check` 全量；更新 `.trellis/spec/backend/*`（新 sync 模块、命令/事件、安全约束）
- [ ] 防火墙提示文案；手动双机联调（Win↔Win，如有条件 Win↔Mac）

## 风险点 / 回滚
- M5 最重；不通过时可独立回退，不影响 M1–M4。
- 全部功能默认关闭开关，回滚只需删除 `sync/` 与注册点。
- 需要两台机器或两个实例联调；回环测试不能覆盖防火墙与 mDNS 多网卡行为。
