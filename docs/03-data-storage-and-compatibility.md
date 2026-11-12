# 数据存储与兼容性

## 1. 数据根

### 1.1 默认位置

当前所有持久化数据都经由 `src-tauri/src/core/paths.rs` 解析：

```text
<app_local_data_dir>/
  dev/                         # tauri dev
    storage.json               # bootstrap manifest
    db/
      clipboard.db
    resources/
      clipboard-images/
        origin/<hash[..2]>/<hash>.png
        thumbnails/<hash[..2]>/<hash>.png
      app-icons/<hash>.png
      file-icons/<hash>.png
    config/
      settings.json
    state/
      window state files
  prod/                        # release build
    ...
```

macOS 的 `app_local_data_dir` 通常对应 `~/Library/Application Support/<bundle identifier>`，具体路径由 Tauri 解析。决定环境的是 `cfg!(dev)`：开发数据在 `dev/`，发布数据在 `prod/`。

### 1.2 自定义目录

当用户选择自定义数据目录时：

```text
<selected-parent>/EcoPasteData/<dev|prod>/
  .ecopaste-storage.json
  db/
  resources/
  config/
  state/
```

`<app_local_data>/<env>/storage.json` 始终保留为启动锚点，内容包含：

- `version`
- `environment`
- `dataDir`
- `updatedAt`

真实数据根必须有匹配环境的 `.ecopaste-storage.json`。如果自定义目录缺失、identity 不匹配或 manifest 无法读取，启动会回退到默认数据根并重写 manifest。

### 1.3 数据目录迁移

数据目录切换的语义：

1. 拒绝新旧目录互相包含的嵌套关系。
2. 校验目标目录：空目录可写入；已有目录必须有合法 identity。
3. 暂停 watcher。
4. 关闭当前 SQLite pool。
5. 复制 `STORAGE_CONTENT_DIRS` 到目标目录。
6. 写入目标 `.ecopaste-storage.json`。
7. 更新 bootstrap manifest。
8. 重新打开数据库。
9. rebase 图片/图标 store，重载设置。
10. 删除旧目录或默认目录中的数据 payload。

迁移失败时会尝试把 manifest 回滚到旧目录并重新打开数据库。

## 2. SQLite

### 2.1 连接配置

`src-tauri/src/db/init.rs`：

- `create_if_missing(true)`
- `journal_mode = WAL`
- `synchronous = NORMAL`
- `foreign_keys = true`
- 关闭 statement logging
- `sqlx::migrate!("./migrations")`

### 2.2 正式 migration

当前仓库只有：

```text
src-tauri/migrations/0001_init.sql
```

`.trellis/spec/backend/database-and-storage.md` 提到的 `0002_fts.sql` 不存在。FTS5 虚拟表和触发器全部在 `0001_init.sql` 中定义。迁移实现必须以实际文件为准。

### 2.3 Schema

#### `clipboard_groups`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | TEXT PK | UUID |
| `name` | TEXT NOT NULL | 分组名 |
| `icon` | TEXT NOT NULL | 图标标识或导入资源引用 |
| `is_hidden` | INTEGER NOT NULL | 是否隐藏 |
| `sort_order` | INTEGER NOT NULL | 排序 |
| `created_at` / `updated_at` | TEXT NOT NULL | UTC 时间 |

#### `clipboard_apps`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | TEXT PK | macOS bundle id；Windows exe 绝对路径 |
| `name` | TEXT NOT NULL | 应用名 |
| `icon_file` | TEXT NULL | `<hash>.png` |
| `platform` | TEXT NOT NULL | `macos` / `windows` |
| `created_at` / `updated_at` | TEXT NOT NULL | UTC 时间 |

#### `clipboard_items`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | TEXT PK | UUID |
| `kind` | TEXT NOT NULL | `text` / `image` / `files` |
| `sub_kind` | TEXT NULL | `rtf` / `html` / `url` / `email` / `color` / `path` |
| `group_id` | TEXT NULL | FK -> `clipboard_groups.id`，删除置 NULL |
| `source_app_id` | TEXT NULL | FK -> `clipboard_apps.id`，删除置 NULL |
| `content` | TEXT NOT NULL | 文本源 / 图片文件名 / 文件路径列表 |
| `content_hash` | TEXT NOT NULL | `blake3("<kind>:<content>")` |
| `search_text` | TEXT NULL | 文本/文件名搜索内容 |
| `summary` | TEXT NULL | 文本摘要，实际截断 256 字符 |
| `file_types` | TEXT NULL | 文件类型串，如 `d,f,f` |
| `size` | INTEGER NULL | 内容字节数或图片大小 |
| `width` / `height` | INTEGER NULL | 图片尺寸 |
| `use_count` | INTEGER NOT NULL DEFAULT 1 | 去重/复用次数 |
| `is_favorite` | INTEGER NOT NULL DEFAULT 0 | 收藏 |
| `is_pinned` | INTEGER NOT NULL DEFAULT 0 | 置顶 |
| `is_sensitive` | INTEGER NOT NULL DEFAULT 0 | 敏感内容标记 |
| `platform` | TEXT NOT NULL | macos/windows |
| `note` | TEXT NULL | 备注 |
| `created_at` / `updated_at` | TEXT NOT NULL | UTC 时间 |

约束：

```sql
CHECK (kind <> 'files' OR trim(content) <> '')
```

索引：

- `idx_clipboard_items_content_hash`
- `idx_clipboard_items_source_app_id`

#### `file_type_icons`

| 字段 | 说明 |
| --- | --- |
| `cache_key` | 文件类型/扩展名缓存键 |
| `platform` | 平台 |
| `icon_file` | 图标文件名 |
| `created_at` / `updated_at` | 时间 |

复合主键：`(cache_key, platform)`。

#### `clipboard_items_fts`

```sql
CREATE VIRTUAL TABLE clipboard_items_fts USING fts5(
    search_text,
    note,
    content='clipboard_items',
    content_rowid='rowid',
    tokenize='trigram'
);
```

三个 trigger：

- AFTER INSERT
- AFTER DELETE
- AFTER UPDATE

Trigger 维护 external-content FTS 的 delete/insert 语义。`UPDATE` 必须先 delete 旧行再 insert 新行。

### 2.4 时间格式

- Rust 模型使用 `DateTime<Utc>`。
- sqlx/SQLite 以 TEXT 存储，实际探测到的格式带微秒和显式 UTC 偏移，例如：

  ```text
  2026-11-12T13:43:15.414023+00:00
  ```

- 这不是 RFC3339 的 `Z` 后缀形式。`storage.json.updatedAt` 走 serde/chrono，单独使用 `Z` 结尾；两套字符串不要混用或互相覆盖。
- 旧版 `history.createTime` 使用 `%Y-%m-%d %H:%M:%S`，按本地时区解析后转 UTC。

Go 端必须保证：

- 读取旧库时不把时间字符串误当 Unix 时间。
- 写入新记录时保持与现有 TEXT 可比较格式一致，至少保留 UTC 偏移语义；不要改成无时区的本地时间。
- 比较排序、保留清理和 FTS 不受时区偏移影响。

## 3. 搜索语义

### 3.1 分词规则

```text
keyword trim 为空
  -> 无搜索条件

所有空白分词字符数 >= 3
  -> FTS5 MATCH

存在 1–2 字符分词
  -> LIKE
```

注意：门槛按“每个空白分词”判断，不是整串字符数。例如 `a b` 两个词都短，必须 LIKE；否则 FTS 对每个短 token 都是 0 命中，AND 语义导致整条查询为空。

### 3.2 FTS 表达式

`foo bar` -> `"foo"* "bar"*`

- 每个 token 用双引号包住。
- token 内的 `"` 转义为 `""`。
- 空白分隔 token 之间默认 AND。

### 3.3 LIKE 兜底

- `%<escaped>%`
- 同时匹配 `search_text` 和 `note`。
- 转义 `\`、`%`、`_`，SQL 使用 `ESCAPE '\'`。

### 3.4 排序与分页

```sql
ORDER BY clipboard_items.is_pinned DESC, <sort>
```

`sort`：

- `createdAtDesc`: `created_at DESC`
- `updatedAtDesc`: `updated_at DESC, created_at DESC`
- `useCountDesc`: `use_count DESC, created_at DESC`

分页使用 `LIMIT ? OFFSET ?`。`has_more = offset + returned < total`。

## 4. 资源文件

### 4.1 剪贴板图片

```text
resources/clipboard-images/origin/<hash[..2]>/<hash>.png
resources/clipboard-images/thumbnails/<hash[..2]>/<hash>.png
```

- 文件名是 PNG 字节的 blake3。
- 原图在采集时写入；已存在则跳过（幂等）。
- 缩略图在读取时懒生成，最长边 300。
- 删除记录时删除原图和缩略图。
- 分片目录为空时尽力删除。

### 4.2 应用图标

```text
resources/app-icons/<hash>.png
```

图标文件名同样来自字节哈希。`clipboard_apps.icon_file` 只存文件名。

### 4.3 文件图标

```text
resources/file-icons/<hash>.png
```

与 `file_type_icons` 表配合，缓存不同文件类型的图标。

### 4.4 兼容要求

Go 版本不应改变：

- 图片记录的 `content` 文件名格式。
- 原图/缩略图相对路径。
- 应用/文件图标目录。
- 图片去重语义。

否则旧数据库中的文件名无法解析，备份覆盖导入后也会出现图片丢失。

## 5. 设置持久化

### 5.1 文件与写入

```text
<data_root>/config/settings.json
```

`SettingsStore`：

- 结构性 camelCase JSON。
- 所有结构体 `serde(default)`。
- 缺字段用默认值补齐，保证旧配置可读。
- 更新使用深度 JSON patch。
- 写入采用临时文件 + rename 原子替换。

### 5.2 设置分组

| 分组 | 关键字段 |
| --- | --- |
| `general` | autoStart、runAsAdmin、trayIcon、dockIcon |
| `appearance` | theme、language |
| `shortcuts` | openClipboard、openPreference、winV |
| `onboarding` | completed、lastStep、legacyImport |
| `update` | autoCheck、frequency、includeBeta、includeNightly、lastCheckedAt、skippedVersion |
| `clipboard.capture` | 类型开关、大小上限、采集顺序 |
| `clipboard.content` | 自动粘贴、中键、plain、隐藏、文件路径、删除规则、自动收藏、复用刷新、排序、项动作 |
| `clipboard.display` | 文本行数、图片高度、文件数量 |
| `clipboard.sensitive` | collectSecrets、redactSecrets |
| `clipboard.filters` | excludedAppIds |
| `clipboard.history` | retention、maxCount、cleanupIntervalHours |
| `clipboard.search` | defaultFocus、clearOnHide |
| `clipboard.window` | position、scrollToTopOnOpen、打开时选中范围/分类/分组、lightweightMode、idleDestroySeconds |
| `clipboard.preview` | hoverEnabled、hoverDelayMs、spaceEnabled |
| `clipboard.feedback` | copySound |

### 5.3 默认值中需要保留的行为

- 采集顺序：files、image、html、rtf、text。
- 文本上限 4 MB，图片上限 100 MB。
- 默认排除 macOS 钥匙串访问和“密码”App。
- `collectSecrets = true`，`redactSecrets = true`。
- `autoPaste = DoubleClickPaste`。
- `autoCheck = false`，频率 daily。Rust `Update` 使用 `derive(Default)`，源码默认不自动检查更新；用户已保存的配置可能为 `true`，不要把本机配置当作源码默认值。
- `copySound = false`。
- 主窗口跟随光标，隐藏后 lightweight/dormant 开启。

## 6. 缓存与状态文件

- `state/` 保存窗口几何等本机运行状态。
- 窗口状态不是业务数据，可以重建，但迁移时应尽量保留体验。
- 资源缓存可在偏好页清理；清理只删除未引用或失效文件，不删除数据库记录。

## 7. 旧数据库兼容策略

### 7.1 不能做的事

1. 不能把 `0001_init.sql` 直接对旧库重放。
2. 不能假设 `_sqlx_migrations` 不存在。
3. 不能重建 `clipboard_items` 而丢失 rowid 与 FTS 关联。
4. 不能删除或重建 FTS 表导致 trigram 索引与内容不同步。
5. 不能把 `content_hash` 改成大小写、编码或哈希算法不同的值。

### 7.2 推荐启动探测

Go 端首次打开数据库时应只读检查：

```text
PRAGMA user_version
PRAGMA journal_mode
PRAGMA foreign_keys
SELECT name, type, sql FROM sqlite_master
SELECT * FROM _sqlx_migrations
PRAGMA table_info(clipboard_items)
PRAGMA index_list(clipboard_items)
```

探测时注意：

- `PRAGMA foreign_keys` 是连接级设置。用独立 `sqlite3` 连接探测到 `0`，不代表应用运行时没有开启；Go 端必须在每个连接上执行 `PRAGMA foreign_keys = ON`。
- 当前真实库的 `PRAGMA user_version` 为 `0`。EcoPaste 的 schema 身份记录在 `_sqlx_migrations`，不能用 `user_version` 判断是否已初始化。
- 真实库已观察到 `journal_mode=wal`、`_sqlx_migrations` 有 `version=1, success=1`、`clipboard_items` 与 `clipboard_items_fts` 行数一致。探测结果只读，不要对现有库重放 `0001_init.sql`。

然后判断：

- 全新空库：建立 Go 自有 schema/migration。
- 已有 EcoPaste 当前 schema：接管，不重建。
- 只有旧版 `history` 表：交给旧版导入流程。
- 未知 schema：只读拒绝写入，提示备份，不自动“修复”。

### 7.3 Schema 版本账本

建议 Go 端使用独立 ledger，例如 `ecopaste_schema_migrations` 或 `PRAGMA user_version`，但必须与 `_sqlx_migrations` 共存：

- 首个 Go 版本识别当前 `0001` 等价 schema 后，写入 Go ledger 基线。
- 后续 Go migration 只做增量变更。
- 不删除 `_sqlx_migrations`，以免旧版本回退时无法识别。

### 7.4 首次迁移安全网

首次用 Go 版本打开旧数据时，应在任何写入前：

1. 复制 `clipboard.db`、`-wal`、`-shm`（如果存在）。
2. 复制 `resources/`。
3. 复制 `config/settings.json`。
4. 生成迁移前备份包或至少生成同目录 timestamp 备份。
5. 在 UI 中暴露恢复路径。

## 8. Go SQLite 驱动评估

当前不指定最终驱动。候选：

| 候选 | 优点 | 风险 |
| --- | --- | --- |
| `modernc.org/sqlite` | 纯 Go，无 cgo，部署简单 | 需验证 FTS5 trigram、WAL、busy timeout、性能 |
| `ncruces/go-sqlite3` | 基于 SQLite WASM，纯 Go 路线 | 需验证 API、FTS5 编译选项和打包 |
| cgo SQLite | 行为最接近 sqlx/libsqlite3 | 破坏无 cgo 构建，增加签名/打包/交叉编译复杂度 |

必须用真实 EcoPaste 旧库做 spike：

1. 打开 WAL 数据库，确认能读 `_sqlx_migrations`。
2. 执行 FTS5 trigram 查询，验证 3 字符、CJK、大小写、前缀和中英文混合。
3. 验证 1–2 字符 LIKE 降级。
4. 验证 external-content trigger 在 INSERT/UPDATE/DELETE 后索引同步。
5. 验证 `PRAGMA foreign_keys = ON`。
6. 验证并发读、单写、busy timeout 和 WAL checkpoint。
7. 验证时间 TEXT 的读取和排序。
8. 验证在 Go 进程内备份/复制数据库时的 WAL 一致性。

## 9. 兼容性优先级

| 优先级 | 数据 |
| --- | --- |
| P0 | `clipboard.db` schema、记录 ID、content_hash、时间字段、FTS5 |
| P0 | `resources/clipboard-images` 文件名与目录 |
| P0 | `.ecopastebak` 明文 ZIP 和加密容器 |
| P1 | `settings.json` 字段与默认值 |
| P1 | `clipboard_apps`、`clipboard_groups`、`file_type_icons` |
| P2 | 窗口状态、缓存文件和日志 |

## 10. 已发现的文档漂移

- `ClipboardItem.summary` 注释写“最多 512 字符”，实际常量 `SUMMARY_MAX_CHARS` 为 256。
- `clipboard/ingest.rs` 顶部注释曾写图片文件名为 `<sha256>.png`，实际 `ImageStore` 使用 PNG 字节的 blake3。
- `.trellis/spec` 提到 `0002_fts.sql`，实际仓库只有 `0001_init.sql`。
- 旧版导入只覆盖历史记录，不应被描述成完整设置/数据迁移。

这些漂移不影响当前产品运行，但 Go 端实现和后续文档必须以源码常量和实际 migration 为准。
