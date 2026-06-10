# TODO: 内容访问密码保护

## 目标

允许用户给单条剪贴板内容设置访问密码。锁定条目在列表中默认遮蔽，输入正确密码后才能查看、复制或粘贴。

## 范围

- 密码与备注独立。
- MVP 仅做命令层访问门禁，不做磁盘内容加密。
- 前端不得接收密码哈希。
- 解锁缓存仅当前前端会话有效，关窗或刷新后失效。

## 数据层

- `clipboard_items` 增加 `password_hash TEXT NULL` 与 `password_hint TEXT NULL`。
- 当前版本仍是 `0.6.0-beta.3`，按项目约定直接改 `0001_init.sql`，不新增迁移文件。
- Rust 模型对前端只暴露 `is_locked` 与 `password_hint`，`is_locked` 由 `password_hash IS NOT NULL` 推导。

## Rust 命令

- `set_item_password(id, password, hint)`：空密码视为清除密码。
- `verify_item_password(id, password) -> bool`：成功返回 true，失败返回 false。
- `read_locked_item(id, password)`：校验后返回完整内容 payload，失败返回专门错误。
- `paste_clipboard_item` 和复制命令遇到锁定条目时必须要求已验证路径，不能绕过。

## 前端

- 卡片操作区增加锁定/解锁入口。
- 锁定条目内容区显示占位、提示和解锁入口。
- 复制、粘贴、备注等敏感入口在锁定未解锁时先弹密码框。
- `clipboardViewStore` 可保存 `unlockedIds`，仅内存态。
- i18n 补齐中英文密码设置、验证、错误、清除等文案。

## 安全注意

- 使用 `argon2` crate 保存密码哈希。
- 对同一条目连续失败做简单内存限流，例如 5 次失败后 30 秒内拒绝。
- 文案必须明确：MVP 是访问保护，不是磁盘加密。

## 验收

- 锁定条目列表不泄露正文、HTML、文件路径等敏感字段。
- 正确密码可临时查看、复制、粘贴，错误密码不会泄露内容。
- 清除密码后条目恢复普通行为。
- 备注和密码互不覆盖。

