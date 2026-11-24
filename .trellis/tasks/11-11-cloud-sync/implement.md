# Implementation Plan

## M1 服务端核心（EcoPasteServer/server）
- [ ] 仓库脚手架、便携工具链（同 MediaHub `tools/`、`env.ps1`）、`go test` 基线
- [ ] `store`（SQLite 迁移）、`auth`（argon2id、令牌哈希、限流 / 锁定）、`tlsx`（自签 + 指纹）
- [ ] `api`：login / items / blobs / history；`hub`：WS 实时 + `since` 补发 + 设备吊销断线
- [ ] `retention`：条数 / 天数 / 容量清理，blob 回收；文件 `FetchFile` 经 WS 中转
- [ ] 单测 + 端到端测试（两个 WS 客户端互通、限流、吊销）

## M2 管理程序（desktop）
- [ ] Tauri 2 脚手架 + 本机 admin API 对接
- [ ] 页面：概览 / 账号 / 设备 / 存储与保留 / 日志 / 设置（端口、证书、指纹、更新）
- [ ] 托盘、开机自启、一键复制连接信息

## M3 客户端 cloud 模块（EcoPaste）
- [ ] `settings.cloud` + `cloud/` 模块（client / session / sync）、指纹固定
- [ ] 与 LAN 同步并存：去重、防回环、`fetch_file_range` 抽象
- [ ] 命令 / 事件常量两端同步；`commands/cloud.rs`
- [ ] 前端「云同步」分组 + 指纹确认弹窗 + 状态；zh-CN / en-US
- [ ] 集成测试（进程内起真实 Go 服务端或其 Rust 替身）

## M4 打包发布
- [ ] EcoPasteServer 便携包 + 更新包 + 校验和 → GitHub Release；自动更新联调
- [ ] EcoPaste 新版本（签名 + latest.json）发布并本机升级
- [ ] 更新 `.trellis/spec`（cloud 协议、安全约束）

## 验证
- 服务端：`go test ./...`；客户端：`cargo clippy -- -D warnings`、`cargo test`、`pnpm lint`
- 手动：公网映射端口后，两台不同网络设备联调；限流 / 吊销 / 指纹不匹配 / 保留策略

## 风险点 / 回滚
- 每个里程碑独立可发布；M3 默认关闭开关，回滚只需移除 `cloud/` 与注册点。
