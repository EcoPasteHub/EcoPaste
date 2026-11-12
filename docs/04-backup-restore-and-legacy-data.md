# 备份、恢复与旧版数据

## 1. 备份文件

### 1.1 扩展名与关联

| 项目 | 值 |
| --- | --- |
| 扩展名 | `.ecopastebak` |
| MIME | `application/vnd.ecopaste.backup` |
| macOS UTI | `com.ayangweb.eco-paste.backup` |
| 角色 | Editor |

macOS 通过 `RunEvent::Opened` 接收文件；Windows 通过命令行参数、second-instance 和拖放接收。偏好窗口被空闲销毁时，事件进入 pending slot，窗口重建后前端主动 `take_pending_backup` 拉取，避免事件丢失。

### 1.2 明文格式

明文 `.ecopastebak` 是标准 ZIP，使用 deflate。内容：

```text
manifest.json
db/clipboard.db
resources/**
config/settings.json
```

`resources/**` 保留原相对目录，例如：

```text
resources/clipboard-images/origin/<hash[..2]>/<hash>.png
resources/app-icons/<hash>.png
resources/file-icons/<hash>.png
```

`manifest.json` 字段：

| 字段 | 说明 |
| --- | --- |
| `formatVersion` | 当前为 1 |
| `appName` | EcoPaste |
| `appVersion` | 导出时版本 |
| `exportedAt` | UTC 时间 |
| `platform` | macos/windows |
| `encryption` | `none` / `password`（Rust 枚举 `rename_all = "camelCase"` 后的实际值） |
| `itemCount` | 总记录数 |
| `textCount` | 文本记录数 |
| `imageCount` | 图片记录数 |
| `filesCount` | 文件记录数 |
| `resourceBytes` | 资源总字节数 |

### 1.3 加密容器

加密备份不是 ZIP 外包一层加密，而是 EcoPaste 自有容器：

```text
12 bytes   magic: "ECOPASTEBAK1"
4 bytes    little-endian JSON header length
N bytes    JSON header (UTF-8, camelCase)
remaining  XChaCha20-Poly1305 ciphertext + 16-byte tag
```

Header：

```json
{
  "formatVersion": 1,
  "mode": "encrypted",
  "kdf": {
    "algorithm": "argon2id",
    "memoryKib": 65536,
    "timeCost": 3,
    "parallelism": 1,
    "salt": [16 bytes]
  },
  "cipher": {
    "algorithm": "xchacha20poly1305",
    "nonce": [24 bytes]
  }
}
```

固定参数：

- Argon2id
- version 0x13
- memory 64 MiB
- time cost 3
- parallelism 1
- output key 32 bytes
- salt 16 bytes
- nonce 24 bytes
- XChaCha20-Poly1305

兼容实现必须保留：

1. magic 的精确 12 字节。
2. header length 的小端 `u32`。
3. header JSON 的字段名和值。
4. salt/nonce 长度。
5. Argon2 参数。
6. XChaCha20-Poly1305 的 nonce 与 tag 处理。
7. ciphertext 紧跟 header 的顺序。

### 1.4 检测逻辑

读取文件开头：

- `PK` -> 按明文 ZIP 处理。
- `ECOPASTEBAK1` -> 按加密容器处理。
- 其它 -> 报“不是有效的 EcoPaste 备份文件”。

加密备份必须提供密码；错误密码或损坏统一报“备份密码不正确或文件已损坏”，不泄漏内部错误。

### 1.5 导出安全

- 先写临时文件。
- 写完后 sync。
- 再原子 persist 到目标路径。
- 资源目录可选；不存在时跳过。
- 目标路径与源文件同一路径时跳过该文件，避免自包含。

## 2. 导入流程

### 2.1 检查

导入前会：

1. 识别明文/加密。
2. 解密到内存 payload。
3. 解压到临时目录。
4. 校验 `manifest.json`。
5. 校验 `db/clipboard.db`。
6. 校验 `config/settings.json`。
7. 返回容器模式、计数、路径等元数据。

### 2.2 Merge

Merge 的语义是“保留现有数据，尽量补齐备份中的内容”：

- 以只读连接打开备份数据库。
- `clipboard_groups`：`INSERT OR IGNORE`，按 id 去重。
- `clipboard_apps`：`INSERT OR IGNORE`，按 id 去重。
- `file_type_icons`：`INSERT OR IGNORE`，按 `(cache_key, platform)` 去重。
- `clipboard_items`：按 `kind + content_hash` 检查当前库是否已有。
  - 已有：跳过，`skippedItems += 1`。
  - 没有：插入，保留原 id、时间、收藏、置顶、备注、敏感标记和来源应用。
- 资源：只复制目标目录中不存在的文件，不覆盖现有文件。
- 设置：读取备份 settings 作为 patch 合并到当前设置。

Merge 不会把已有记录替换成备份版本，也不会合并 `use_count` 或切换 `updated_at`。

### 2.3 Overwrite

Overwrite 是“用备份数据替换当前数据”：

1. 暂停 watcher。
2. 关闭当前 SQLite pool。
3. 将备份数据库直接复制到 live `clipboard.db`，再删除 `-wal` / `-shm`。
4. 重新 `db::init`。
5. 替换 `resources/`。
6. 刷新 AppsRegistry。
7. 替换 settings。
8. 发剪贴板更新事件。

当前实现不是原子替换：

- 数据库复制没有写临时文件后 rename，也没有保留旧库快照。
- 复制成功但重新打开失败、或资源目录替换中途失败时，没有自动回滚。
- `-wal` / `-shm` 在复制后直接删除，旧连接必须已完全关闭，否则可能丢失尚未 checkpoint 的数据。
- 设置在数据库和资源替换之后才替换；前序步骤失败时，设置不会回滚。
- 资源目录替换与数据库替换不是同一个事务。

因此 Go 版本若要加强可靠性，应新增“导入前快照 + 临时文件 + 原子替换 + 失败恢复”，不能把当前 Rust 实现视为已具备原子性。

### 2.4 Merge 与 overwrite 的差异

| 行为 | Merge | Overwrite |
| --- | --- | --- |
| 现有记录 | 保留 | 替换 |
| 备份记录去重 | kind + content_hash | 不适用 |
| 资源 | 缺什么补什么 | 替换目录 |
| 设置 | patch 合并 | 用备份文件替换 |
| 数据库连接 | 继续使用 | 关闭并重开 |
| 用户操作风险 | 低 | 高，必须有确认 |

## 3. 旧版 EcoPaste 数据

### 3.1 搜索候选目录

macOS：

```text
~/Library/Application Support/com.ayangweb.EcoPaste
~/Library/Application Support/EcoPaste
```

Windows：

```text
%APPDATA%/com.ayangweb.EcoPaste
%APPDATA%/EcoPaste
```

还会读取旧 `.store.json` / `.store.dev.json` 中的：

```text
globalStore.env.saveDataDir
```

把自定义旧数据目录加入候选。

### 3.2 旧数据库文件

历史文件名包括：

- `.store.db` / `.store.dev.db`
- `.window-state.db` / `.window-state.dev.db`
- `EcoPaste.db` / `EcoPaste.dev.db`
- `EcoPaste.v2.db` / `EcoPaste.v2.dev.db`

实际检测并非“同时保留其它环境候选”，而是只接受当前环境的固定文件名：

| 环境 | 固定候选 |
| --- | --- |
| prod | `.store.db`、`.window-state.db`、`EcoPaste.db`、`EcoPaste.v2.db` |
| dev | `.store.dev.db`、`.window-state.dev.db`、`EcoPaste.dev.db`、`EcoPaste.v2.dev.db` |

其它环境的固定文件名会被显式排除。prod 还会接受文件名中不含 `.dev.` 的 `.sqlite` / `.sqlite3` / `.db` 文件；dev 不扫描这些泛化数据库文件。多个目录/数据库候选按可导入记录数选择最大者；数量相同时保留先扫描到的候选，不是简单取第一个命中。

### 3.3 旧 `history` 表

旧表字段：

```text
id
type
value
favorite
createTime
note
width
height
```

映射：

| 旧字段 | 新语义 |
| --- | --- |
| `type = text` | 纯文本 payload |
| `type = files` | `value` 为 JSON 路径数组 |
| `type = image` | `value` 为图片文件名，拼接到旧 `images/` 目录 |
| `favorite` | `is_favorite` |
| `createTime` | 按本地时区解析为 UTC |
| `note` | 备注 |
| `width` / `height` | 图片尺寸 |

旧图片路径：

```text
<legacy-data-dir>/images/<value>
```

图片会被读入后重新走当前 `ImageStore`，因此目标文件名会变成当前 PNG 字节哈希，而不是保留旧文件名。

### 3.4 导入行为

- 用户选择普通条目和/或收藏条目。
- 按 `createTime ASC, id ASC` 读取。
- 逐条构造 payload 并走 `build_item_with_settings`。
- 新 id 使用 UUID。
- `is_pinned = false`。
- `source_app_id = null`。
- 命中当前 content_hash 去重时跳过。
- 导入完成后更新 onboarding 状态并发事件。

### 3.5 不导入的内容

当前旧版导入只覆盖历史记录，不导入：

- 旧设置。
- 旧快捷键。
- 旧窗口位置。
- 旧分组。
- 旧应用来源元数据。
- 旧收藏以外的 UI 状态。

因此目标版本不能把“旧版数据导入”描述为“完整数据迁移”。

## 4. 兼容性验收用例

### 4.1 备份

1. 明文备份导出/导入。
2. 加密备份导出/导入。
3. 错误密码。
4. 截断 magic、错误 header length、非法 JSON、错误 salt/nonce 长度。
5. 备份内有图片、应用图标、文件图标。
6. 同一数据库重复 merge，第二次全部 skipped。
7. Merge 保留现有记录和现有资源。
8. Overwrite 后图片路径可解析、列表可搜索、备注仍可搜索。
9. 备份文件从 Finder 打开、拖放打开、second instance 打开。
10. Overwrite 在中途失败时仍能回到导入前状态（Go 版本的目标要求；当前 Rust 实现没有完全满足）。

### 4.2 旧版数据

1. 默认旧目录检测。
2. 自定义 `saveDataDir` 检测。
3. text/files/image 三类记录。
4. 收藏/普通选择。
5. 缺失图片跳过。
6. 重复内容跳过。
7. 本地时区 `createTime` 转 UTC。
8. 旧 `history` 不存在或 schema 异常时不崩溃。

### 4.3 数据目录

1. 目录迁移后数据库、图片、图标、设置全部可读。
2. 目标 identity 错误时拒绝。
3. manifest 损坏时回退默认目录。
4. 迁移失败回滚。
5. 删除旧数据只删除 EcoPaste 所有目录，不误删用户父目录。

## 5. 对 Go 实现的硬性要求

- 先读后写，未知 schema 不自动改。
- 导入前备份当前数据。
- Merge 按 `kind + content_hash` 去重，不按 id 粗暴覆盖。
- Overwrite 在旧版中不是原子操作；Go 版本必须补上原子替换和失败恢复。
- 备份格式不得改变现有 magic、参数、字段名和路径。
- 旧版导入不能扩大承诺到未实现的设置迁移。
- 所有加密相关参数、nonce 和 tag 处理必须用 Rust 版生成的样本做交叉测试。
