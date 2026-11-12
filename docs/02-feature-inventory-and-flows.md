# 功能清单与业务流程

## 1. 功能全景

### 1.0 当前平台差异

第一阶段以 macOS 为目标，但理解现有实现时需要区分：

| 行为 | macOS | Windows |
| --- | --- | --- |
| 剪贴板监听 | `clipboard-rs` 轮询 `NSPasteboard.changeCount`，间隔 120ms | `WM_CLIPBOARDUPDATE` 事件 |
| 读取失败重试 | 无专门重试延迟 | 15ms、35ms、75ms，最多 4 次尝试 |
| 主窗口 | WebViewWindow 转 NSPanel | `focusable=false` 的 WebView 窗口 |
| 列表项菜单 | 原生 muda 菜单 | 无焦点 WebView 菜单窗口 |
| 模拟粘贴 | `CGEvent` ⌘V | 平台键盘注入 |
| 键盘导航 | AppKit/WebView 默认输入 | 主窗口收不到键时走 Rust `keyboard/` |

### 1.1 剪贴板采集

| 能力 | 规则 |
| --- | --- |
| 文本 | 支持 plain text、HTML、RTF |
| 图片 | 统一存为 PNG，保留宽高和字节数 |
| 文件 | 保存路径列表和目录/文件类型 |
| 采集开关 | text、html、rtf、image、files 可独立关闭 |
| 采集优先级 | 默认 files -> image -> html -> rtf -> text，可由设置调整 |
| 大小限制 | 文本、图片分别配置 MB 上限，0 表示不限 |
| 应用排除 | 命中来源应用排除列表时，不读取、不入库、不发事件 |
| 敏感内容 | 可跳过收录，或收录并标记后在展示时脱敏 |
| 监听暂停 | 托盘可停止/恢复监听，暂停期间不读取也不入库 |
| 手动重读 | “重新读取”走与监听相同的读取、建模、去重、事件链路 |

### 1.2 历史记录与搜索

| 能力 | 规则 |
| --- | --- |
| 列表过滤 | 全部、文本、图片、文件、收藏 |
| 自定义分组 | 分组可创建、重命名、隐藏、排序、删除、导入 SVG 图标 |
| 搜索范围 | `search_text` 与 `note` |
| 长词搜索 | 每个空白分词都 ≥3 字符时走 FTS5 trigram 前缀匹配 |
| 短词搜索 | 存在 1–2 字符分词时降级到 `LIKE`，并转义 `\ % _` |
| 排序 | 创建时间倒序、最近使用倒序、使用次数倒序 |
| 置顶优先 | 所有排序下 `is_pinned DESC` 恒在第一条排序条件 |
| 分页 | 每页数量由前端决定；Rust 返回本页、总数、hasMore |
| 列表裁剪 | 文本列表行不返回完整 `content` / `search_text`，用 `summary` 渲染 |
| 来源应用 | 列表 LEFT JOIN 应用名和图标，命令层补绝对路径 |

### 1.3 条目元数据

| 能力 | 规则 |
| --- | --- |
| 收藏 | 切换 `is_favorite`，不刷新 `updated_at` |
| 置顶 | 切换 `is_pinned`，不刷新 `updated_at` |
| 备注 | 编辑 `note`，不刷新 `updated_at` |
| 分组 | 设置 `group_id`，不刷新 `updated_at` |
| 自动收藏 | 可配置新采集内容自动收藏 |
| 使用次数 | 去重命中或启用 `update_on_reuse` 时增加 |
| 删除保护 | 收藏/置顶可独立控制是否允许删除、是否需要二次确认 |

### 1.4 复用动作

| 动作 | 行为 |
| --- | --- |
| 复制 | 写回剪贴板，不模拟 ⌘V |
| 纯文本复制/粘贴 | 忽略 HTML/RTF，写 plain 表示 |
| 粘贴 | 写回 + 隐藏/让出 key + 模拟 ⌘V |
| 文件粘贴为路径 | 文件记录写回换行分隔路径文本 |
| 打开链接 | 仅 URL 子类型 |
| 发送邮件 | 仅 email 子类型 |
| Finder 中显示 | path 子类型或 files 记录 |
| 保存图片 | 图片记录另存为 PNG |
| 拖出 | 文件、图片、文本拖到外部应用 |
| 自动粘贴 | 单击/双击粘贴、单击/双击复制等模式 |
| 中键动作 | 禁用、粘贴、纯文本粘贴、复制、纯文本复制 |

### 1.5 展示与交互

| 能力 | 规则 |
| --- | --- |
| 列表内容 | 文本摘要、图片缩略图、文件图标/图片预览 |
| 摘要长度 | `SUMMARY_MAX_CHARS = 256` |
| 图片缩略图 | 最长边 300，首次读取时懒生成 |
| 文件条目 | 单图片文件走图片预览，其它走文件列表 |
| 悬停预览 | 可配置开关和 300/500/1000ms 延迟 |
| 空格预览 | 可配置 |
| 原文预览 | HTML/RTF 渲染前可查看原始内容 |
| 复制提示音 | 可配置 |
| 主题 | auto/light/dark |
| 语言 | zh-CN / en-US |

### 1.6 窗口、快捷键与系统集成

| 能力 | 当前实现 |
| --- | --- |
| 唤起剪贴板 | 默认 `Alt+C` |
| 打开偏好 | 默认 `Alt+X` |
| Windows Win+V | 可配置替换系统剪贴板历史面板 |
| 剪贴板窗口位置 | 跟随光标、记忆、居中 |
| 打开时滚动/筛选 | 可配置回到顶部、范围、分类、自定义分组 |
| 窗口 pin | 固定后失焦不自动隐藏 |
| 自动隐藏暂停 | 文件选择、保存图片等原生交互期间暂停 |
| 窗口轻量化 | 主窗口隐藏 5 秒后 dormant；其它窗口空闲可销毁重建 |
| 托盘/菜单栏 | 偏好、停止/开启监听、开源地址、检查更新、版本、重启、退出 |
| Dock 图标 | 可配置 |
| 开机自启 | 可配置 |
| 单实例 | 第二实例默认唤起偏好或 onboarding |
| Dock reopen | 无可见窗口时打开偏好/onboarding |
| 文件关联 | 双击 `.ecopastebak` 交给备份导入流程 |

### 1.7 数据管理与更新

| 能力 | 当前实现 |
| --- | --- |
| 数据目录 | 默认目录或自定义目录 |
| 存储用量 | 数据库、资源、设置和总大小 |
| 目录迁移 | 关闭连接池、复制数据、更新 manifest、重开数据库、rebase stores |
| 缓存清理 | 清理未引用图片、应用图标、文件图标、失效缩略图 |
| 备份导出 | 加密或明文 `.ecopastebak` |
| 备份导入 | merge 或 overwrite |
| 旧版导入 | 从旧 `history` 表导入普通/收藏记录 |
| 更新 | Tauri updater，stable/beta/nightly 通道；daily/weekly/monthly 自动检查；可跳过版本 |

## 2. 启动与关闭流程

### 2.1 启动

```text
main
  -> admin elevation (Windows)
  -> Tauri plugin 初始化
  -> settings::init
       -> 解析 app_data_dir
       -> 读 config/settings.json
       -> 缺字段用 serde default 补全
  -> db::init
       -> open SQLite
       -> WAL / FK / NORMAL
       -> sqlx migration
  -> clipboard::init
       -> WritebackGuard / ImageStore / AppIconStore / FileIconStore
       -> AppsRegistry
       -> cleanup scheduler
       -> watcher thread
  -> shortcut / autostart / tray
  -> macOS NSPanel
  -> onboarding 或主窗口
  -> update scheduler
```

### 2.2 关闭

- 退出前保存所有窗口几何。
- macOS 文件打开事件在 `RunEvent::Opened` 处理，带 panic 兜底。
- 数据库连接由进程退出和 sqlx pool Drop 管理；overwrite 导入和数据目录切换使用显式热替换。

## 3. 剪贴板采集主链路

```text
OS clipboard change
  -> clipboard-rs watcher
  -> capture frontmost app
  -> check excluded app
  -> read settings snapshot
  -> read payload by capture order
  -> build draft
       text: HTML/RTF/plain + sub-kind + summary + search_text
       files: newline paths + file types + basenames
       image: PNG bytes + hash + store origin
  -> sensitive check
  -> WritebackGuard.should_skip(content_hash)
  -> upsert app
  -> upsert item
       existing content_hash -> use_count + 1, updated_at = now
       new -> INSERT
  -> emit clipboard://updated
```

### 3.1 为什么先抓前台应用

监听回调必须在读取 payload 前同步捕获 `NSWorkspace.frontmostApplication`。等到异步任务再查时，前台应用可能已经切换，来源应用会记错。

### 3.2 文本语义

- `content` 存最终源表示：HTML 源、RTF 源或 plain 文本。
- `search_text` 存 OS 同时提供的 plain 文本。
- HTML/RTF 只有在 plain 文本非空时才可能入库。
- `summary` 用 plain 文本 trim 后截断到 256 个 Unicode 标量。
- plain 文本才做 URL/email/color/path 子类型识别。
- `size` 是最终 `content` 的 UTF-8 字节数。

### 3.3 图片语义

- macOS 先尝试 `public.png` 原始字节，从 PNG IHDR 解析宽高。
- 非 PNG（如 TIFF）或 PNG 损坏时才解码并重新编码为 PNG。
- `content` 存图片文件名 `<blake3>.png`。
- 去重哈希针对文件名，文件名来自 PNG 字节的 blake3，因此对字节敏感。
- 原图写入 `resources/clipboard-images/origin/<hash[..2]>/<hash>.png`。
- 缩略图不在复制热路径生成，首次读取时生成到 `thumbnails/<hash[..2]>/<hash>.png`。

### 3.4 文件语义

- `content` 是换行分隔的路径串。
- `file_types` 是 `d` / `f` 逗号串，例如 `d,f,f`。
- `search_text` 是文件名 basename 的换行串。
- `summary` 为 NULL。
- 单文件图片可走图片卡片，多个文件或非图片走文件列表。

### 3.5 去重

`content_hash = blake3("<kind>:<content>")`，kind 前缀避免 text/files 同串误判。

命中已有记录时：

- 不插入新行。
- `use_count += 1`。
- `updated_at = now`。

收藏、置顶、备注、分组等元数据更新不走该逻辑，因此不刷新 `updated_at`。

### 3.6 写回回环抑制

应用自身写回会触发 OS 监听。写回前把内容的 `content_hash` 登记到 `WritebackGuard`：

- TTL 2 秒。
- 监听读到相同 hash 时消费登记并跳过。
- 登记可同时有多个，因为 HTML/RTF 写回同时写 plain fallback。
- 使用 hash 而不是布尔标志，避免吞掉期间的真实复制。

## 4. 列表与搜索流程

### 4.1 列表查询

```text
React query state
  -> list_clipboard_items(query)
  -> Rust build QueryBuilder
  -> keyword filter
       all tokens >= 3 chars -> FTS5 MATCH
       else -> LIKE with ESCAPE
  -> group/kind/favorite/pinned/group_id filters
  -> ORDER BY is_pinned DESC, sort
  -> LIMIT/OFFSET
  -> command layer enrich
       source app icon path
       image thumbnail path
       file entries + file icons
       available actions
       color preview
       display created_at
  -> ClipboardItemPage
```

### 4.2 FTS5 细节

`clipboard_items_fts` 是 external-content FTS5 表：

- 索引列：`search_text`、`note`。
- `content='clipboard_items'`，`content_rowid='rowid'`。
- tokenizer：`trigram`。
- 插入、删除、更新由三个 trigger 同步。

查询表达式把每个 token 双引号包裹并加 `*` 前缀通配；双引号会转义。短词不能走 trigram，必须降级 LIKE，否则 AND 语义会把整条查询变成 0 结果。

### 4.3 前端缓存

`useClipboardItems` 维护 range cache：

- 每页 30。
- 预加载 30。
- `clipboard://updated` 时按 id 增量拉取或重新拉取受影响页。
- 删除、清空、导入后失效缓存并刷新 total。

## 5. 收藏、置顶、备注、分组

### 5.1 收藏

- `toggle_clipboard_item_favorite` 更新 `is_favorite`。
- 收藏不是复制，不刷新 `updated_at`。
- 收藏分组是列表 UI 过滤，不是 `clipboard_groups` 用户分组。

### 5.2 置顶

- `toggle_clipboard_item_pinned` 更新 `is_pinned`。
- 置顶不刷新 `updated_at`。
- 查询排序始终先按 `is_pinned DESC`。

### 5.3 备注

- 备注存 `note`，参与 FTS 搜索。
- 备注更新不刷新 `updated_at`。

### 5.4 自定义分组

- 表 `clipboard_groups`。
- 字段含 `name`、`icon`、`is_hidden`、`sort_order`。
- 删除分组时条目 `group_id` 通过外键 `ON DELETE SET NULL` 置空。
- 支持批量保存排序和主界面显隐。
- 图标支持内置图标或导入 SVG，大小限制 256 KiB。

## 6. 复制、粘贴与窗口时序

### 6.1 复制

```text
write_to_clipboard(id, plain)
  -> load full item
  -> apply copy_plain / force plain
  -> write text/html/rtf/image/files
  -> guard.suppress(content_hash)
  -> optionally mark reuse
  -> optionally hide window
```

### 6.2 粘贴

```text
paste_clipboard_item(id, plain)
  -> load full item
  -> resolve plain mode:
       text + paste_plain
       files + paste_files_as_path
  -> write clipboard + suppress hash
  -> mark reuse if enabled
  -> if pinned: resign key (macOS)
     else: hide window
  -> sleep 50ms
  -> CGEvent Cmd+V
  -> if pinned: sleep 50ms + make key again
```

这两个 50ms 延迟不是装饰：

- `hide_window` / `resign_key` 是主线程异步派发，必须等 AppKit 真正 order out 或让出 key。
- 固定窗口中 NSPanel 仍是 key window 时会吞掉 ⌘V。
- 粘贴后立即抢回 key 会吞掉目标 App 的按键。

### 6.3 自动粘贴模式

`AutoPaste` 支持：

- Disabled
- SingleClickPaste
- DoubleClickPaste（默认）
- SingleClickCopy
- DoubleClickCopy

中键支持独立动作枚举。平台差异由前端普通点击/中键事件和 Rust 命令共同处理。

## 7. 预览

- `clipboard-preview` 是独立 `DestroyWhenIdle` 窗口。
- 预览由锚点 rect、scale factor、work area、主窗口 rect 计算 placement。
- placement 可为 right/left/bottom/top。
- 预览 payload 包含文本、图片路径、尺寸、文件条目、敏感标记、存在性。
- 敏感内容是否脱敏由 Rust 侧决定，前端不自行判断。
- 预览窗口在剪贴板窗口隐藏时被抑制/关闭。

## 8. 删除与清理

### 8.1 删除单条

- 图片记录删除时同时删除原图和缩略图。
- 收藏/置顶是否允许删除、是否二次确认由设置控制。
- 收藏条目可限制为只能在收藏分组中删除。

### 8.2 清空

- `clear_items(delete_favorites, delete_pinned)`。
- 未显式删除的收藏/置顶保留。
- 返回被删图片文件名，供资源清理。

### 8.3 自动清理

- 时间保留：`created_at < cutoff`。
- 最大条数：在非收藏/非置顶集合内按 `created_at DESC` 保留前 N 条。
- 收藏和置顶永远受保护。
- 清理周期以小时配置，0 表示关闭周期清理，但启动时仍清理一次。

## 9. 备份与恢复流程

```text
export
  -> collect DB/resources/settings
  -> build manifest
  -> ZIP deflate
  -> optional Argon2id + XChaCha20-Poly1305
  -> write temp + sync + atomic persist

import inspect
  -> detect plain ZIP or encrypted container
  -> validate manifest/db/settings
  -> return mode/path counts

import merge
  -> pause watcher
  -> read-only backup DB
  -> INSERT OR IGNORE groups/apps/file_type_icons
  -> dedupe items by kind + content_hash
  -> copy missing resources
  -> merge settings

import overwrite
  -> pause watcher
  -> close pool
  -> replace live DB
  -> reopen DB
  -> replace resources
  -> refresh apps registry
  -> replace settings
```

详细字节格式见 [04-backup-restore-and-legacy-data.md](04-backup-restore-and-legacy-data.md)。

## 10. 窗口与托盘行为

### 10.1 显示

- 主剪贴板窗口在 macOS 是 NSPanel，显示前延迟 16ms。
- 显示后切换 collection behavior 到 `can_join_all_spaces`。
- 非剪贴板窗口恢复上次几何后普通 show/focus。
- 隐藏前保存窗口状态。

### 10.2 生命周期

| 窗口 | 保留策略 |
| --- | --- |
| clipboard | Permanent |
| preference | DestroyWhenIdle |
| onboarding | DestroyWhenIdle |
| update | DestroyWhenIdle |
| clipboard-preview | DestroyWhenIdle |
| context menu/submenu (Windows) | DestroyWhenIdle |

- 主窗口隐藏 5 秒后进入 dormant。
- 非主窗口空闲 60 秒（可配置，夹在 5 秒到 24 小时）销毁。
- 销毁前广播 `window://before-destroy`，留 500ms 给前端保存或 keepalive。
- dirty owner 和 keepalive 可阻止销毁。

## 11. 当前实现与目标实现的对应关系

| 当前能力 | 目标层 |
| --- | --- |
| React 列表和键盘 | AppKit `NSTableView` + delegate/responder |
| React 设置表单 | AppKit 偏好窗口控制器 |
| Tauri command | Go service/controller 方法 |
| Tauri event | Go 观察者/notification/delegate |
| sqlx 仓储 | Go repository + SQLite |
| `clipboard-rs` | Go 剪贴板适配层 |
| Tauri updater | 独立 update 适配层，不能假定兼容 |
| NSPanel 插件 | AppKit 自建 panel 或 nsgo `desktop.OpenPanel` 外壳 |

迁移时应围绕功能语义和数据库契约设计，不应围绕 React 组件结构设计。

