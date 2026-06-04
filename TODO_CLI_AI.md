# EcoPaste CLI / AI Agent Roadmap

> 本文件独立于 `TODO.md`，专门记录 EcoPaste CLI、AI Tool Manifest、Skill 文档与 MCP 集成方向。
> 目标不是替代桌面端，而是在 Rust 核心能力之上提供稳定、可脚本化、可被 AI Agent 调用的本地数据访问接口。

## 1. 目标

让 EcoPaste 在 GUI 之外提供一组稳定 CLI 能力，使用户和 AI 工具可以直接读取本地剪贴板数据：

```bash
ecopaste latest
ecopaste search "React"
ecopaste get 12345
ecopaste notelist --date 20250604
```

进一步提供标准化能力描述文档，让 Codex、Claude Code、Cursor、Gemini CLI 等工具能够理解并安全调用 EcoPaste：

- CLI 手册
- JSON Tool Manifest
- Codex / Claude Code / Cursor 友好的 Skill 文档
- 可选 MCP Server

## 2. 架构判断

当前项目整体适合向 CLI / AI Agent 方向扩展，原因是核心数据已经基本下沉到 Rust：

- 数据库读写在 `src-tauri/src/db/`
- 数据模型在 `src-tauri/src/db/models.rs`
- 剪贴板入库与识别在 `src-tauri/src/clipboard/`
- GUI 命令只是通过 Tauri command 调用 Rust 能力

但后续不建议让 CLI 直接调用 `commands/` 层。`commands/` 层面向 Tauri IPC，带有 `AppHandle` / `State<SqlitePool>` / 前端交互语义，不适合作为 CLI 核心 API。

推荐演进方向：

```text
src-tauri/src/
  core/
    paths.rs
  db/
    items.rs
    models.rs
  services/
    clipboard_query.rs
    clipboard_note.rs
    clipboard_export.rs
  commands/
    clipboard.rs

src-cli/
  main.rs
  commands/
    latest.rs
    search.rs
    get.rs
    notelist.rs
```

长期可拆成 workspace：

```text
crates/
  ecopaste-core/
  ecopaste-tauri/
  ecopaste-cli/
  ecopaste-mcp/
```

第一阶段不强制拆 workspace，优先用最小结构验证可行性。

## 3. 核心原则

- CLI 复用 Rust 核心逻辑，不重新实现查询规则。
- CLI 默认只读，写操作后置。
- CLI 输出必须支持 `--json`，方便 AI Agent 稳定解析。
- 默认输出面向人类可读，`--json` 面向工具调用。
- CLI 不依赖 Tauri `AppHandle`。
- 数据库路径解析必须从 Tauri 中抽离，支持 GUI 与 CLI 共用。
- 所有可被 AI 调用的命令需要稳定退出码与错误 JSON。
- AI 文档先描述只读能力，避免早期开放高风险写操作。

## 4. 阶段拆分

### 阶段 1：抽离可复用查询核心

目标：

- 新增不依赖 Tauri command 的查询 service。
- 抽离数据库路径解析，使 CLI 可以直接连接 EcoPaste SQLite。

建议新增：

```text
src-tauri/src/services/
  mod.rs
  clipboard_query.rs
```

核心能力：

- `latest(limit)`
- `search(keyword, limit, offset)`
- `get(id)`
- `notelist(date)`

需要处理：

- DB 路径解析
- SQLite 只读连接
- 日期范围计算
- 输出模型裁剪，避免把过大的 HTML / RTF 原文直接打到终端

验收标准：

- 不启动 Tauri GUI，也能通过 Rust 单测调用查询 service。
- `search` 复用现有 FTS / LIKE 查询逻辑。
- `notelist --date 20250604` 能映射为本地时区当天范围。
- 不改变现有桌面端行为。

复杂度：中低。

风险：

- 当前 DB 路径依赖 `AppHandle`。
- 查询模型需要避免暴露过大的 `content`。
- 日期查询需要明确使用本地时区还是 UTC。

### 阶段 2：实现只读 CLI

目标：

新增 `ecopaste` CLI，只提供只读命令：

```bash
ecopaste latest
ecopaste latest --limit 20 --json
ecopaste search "React"
ecopaste search "React" --limit 10 --json
ecopaste get <id>
ecopaste get <id> --json
ecopaste notelist --date 20250604
ecopaste notelist --date 20250604 --json
```

建议依赖：

- `clap`
- `serde_json`
- 可选 `comfy-table` 或保持纯文本输出

输出规范：

- 默认输出简洁 table / plain text。
- `--json` 输出稳定结构。
- 失败时 stderr 输出错误信息，退出码非 0。
- 后续可加 `--json-error` 输出结构化错误。

示例 JSON：

```json
{
  "items": [
    {
      "id": "xxx",
      "kind": "text",
      "subKind": "html",
      "summary": "固定布局尺寸，只动画 transform/opacity",
      "size": 30,
      "createdAt": "2026-06-04T10:04:00Z",
      "updatedAt": "2026-06-04T10:04:00Z"
    }
  ]
}
```

验收标准：

- CLI 可以读取 GUI 当前使用的数据库。
- `latest/search/get/notelist` 四个命令可用。
- 所有命令支持 `--json`。
- GUI 与 CLI 可以同时运行，CLI 只读不破坏 GUI。

复杂度：中。

风险：

- 打包时 CLI binary 如何随 Tauri app 分发需要后续设计。
- Windows/macOS app data 路径要和 GUI 完全一致。
- GUI 写入时 CLI 读取需要依赖 SQLite WAL 正常工作。

### 阶段 3：AI Tool Manifest 与文档

目标：

提供 AI 工具可读的能力描述，先不实现 MCP。

建议新增：

```text
docs/cli.md
docs/ai-tools.md
docs/ecopaste.tool-manifest.json
docs/skills/ecopaste/SKILL.md
```

文档内容：

- 安装方式
- 命令列表
- 参数说明
- JSON 输出 schema
- 安全边界
- 示例 prompt
- 常见错误

Tool Manifest 示例方向：

```json
{
  "name": "ecopaste",
  "description": "Read local EcoPaste clipboard history through a safe CLI.",
  "commands": [
    {
      "name": "latest",
      "run": "ecopaste latest --json",
      "readOnly": true
    },
    {
      "name": "search",
      "run": "ecopaste search <query> --json",
      "readOnly": true
    }
  ]
}
```

验收标准：

- Codex / Claude Code / Cursor / Gemini CLI 可以根据文档理解如何调用 CLI。
- 文档明确标注只读能力。
- 每个命令都有 JSON 示例。
- JSON 字段命名与实际 CLI 输出一致。

复杂度：低。

风险：

- 不同 AI 工具对 manifest 没有完全统一标准。
- 需要避免文档承诺尚未实现的写操作。

### 阶段 4：MCP Server

目标：

提供标准 MCP tools，让支持 MCP 的客户端直接调用 EcoPaste 能力。

建议 tools：

- `ecopaste_latest`
- `ecopaste_search`
- `ecopaste_get`
- `ecopaste_notelist`

第一版 MCP 可以包装 CLI，降低复杂度：

```text
MCP Client -> ecopaste-mcp -> ecopaste CLI -> SQLite
```

后续再改为直接复用 `ecopaste-core`：

```text
MCP Client -> ecopaste-mcp -> ecopaste-core -> SQLite
```

验收标准：

- MCP 客户端能列出 EcoPaste tools。
- tools 返回结构化 JSON。
- 默认只读。
- 错误可被客户端识别。

复杂度：中到偏高。

风险：

- MCP 协议与 Rust SDK 生态可能变化。
- 需要考虑本地数据隐私与权限边界。
- 需要避免 AI Agent 在无用户意图时读取敏感剪贴板。

### 阶段 5：写操作 CLI

目标：

在只读能力稳定后，谨慎开放写操作：

```bash
ecopaste note set <id> "..."
ecopaste favorite <id>
ecopaste delete <id>
ecopaste copy <id>
```

原则：

- 默认需要明确命令，不做隐式写入。
- 危险操作支持 `--yes`。
- AI manifest 中默认不暴露 delete。
- 写剪贴板与模拟粘贴要区分。

验收标准：

- 修改备注后 GUI 实时或下次刷新可见。
- 删除前有确认机制。
- 写操作失败不会破坏数据库。

复杂度：中。

风险：

- GUI 与 CLI 并发写入。
- AI Agent 误删 / 误改数据。
- 系统剪贴板写入涉及平台权限与监听回环。

## 5. 推荐优先级

建议顺序：

1. 抽离 DB 路径与查询 service。
2. 做只读 CLI：`latest/search/get/notelist`。
3. 固化 JSON 输出 schema。
4. 写 `docs/cli.md` 与 `docs/ecopaste.tool-manifest.json`。
5. 写 Codex / Claude Code Skill 文档。
6. 评估 MCP Server。
7. 最后再考虑写操作。

## 6. 初始命令设计

### `ecopaste latest`

列出最近剪贴板记录。

参数：

- `--limit <n>`：默认 10。
- `--json`：输出 JSON。

### `ecopaste search <query>`

搜索剪贴板记录。

参数：

- `query`：搜索关键词。
- `--limit <n>`：默认 20。
- `--offset <n>`：默认 0。
- `--json`：输出 JSON。

### `ecopaste get <id>`

读取单条记录。

参数：

- `id`：剪贴板记录 ID。
- `--plain`：文本条目只输出纯文本，HTML/RTF 使用 `search_text`。
- `--json`：输出 JSON。

### `ecopaste notelist --date <yyyymmdd>`

列出指定日期有备注的记录，或后续扩展为指定日期剪贴板记录摘要。

参数：

- `--date <yyyymmdd>`：本地日期。
- `--json`：输出 JSON。
- `--with-content`：可选，带上内容摘要。

## 7. 数据模型建议

CLI 输出不建议直接复用完整 `ClipboardItem`，而应定义专用模型：

```rust
pub struct CliClipboardItem {
    pub id: String,
    pub kind: ClipboardKind,
    pub sub_kind: Option<ClipboardSubKind>,
    pub summary: Option<String>,
    pub size: Option<i64>,
    pub note: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}
```

原因：

- 避免默认输出巨大 HTML / RTF / 文件列表。
- 避免把 GUI 专用字段暴露给 CLI。
- 输出 schema 更稳定。

## 8. 安全与隐私

剪贴板历史可能包含敏感内容，AI Agent 集成必须默认保守：

- 默认只读。
- 默认返回摘要，不返回完整内容。
- `get <id>` 才返回完整内容。
- AI manifest 中明确标注本地敏感数据。
- 可考虑后续增加设置项：是否允许 CLI / MCP 访问。
- MCP 写操作默认关闭。

## 9. 总体复杂度结论

只读 CLI：中低复杂度，可较快落地。

AI 文档 / Tool Manifest：低复杂度，依赖 CLI 稳定输出。

MCP Server：中到偏高复杂度，建议在 CLI 稳定后实现。

写操作：中复杂度，需要谨慎处理权限、确认、并发与 GUI 同步。

整体路线可行，且与 EcoPaste 当前 Rust-first 架构一致。关键是先把 Tauri command 与可复用核心 service 分开，让 GUI、CLI、MCP 都调用同一套 Rust 核心能力。
