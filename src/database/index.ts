import Database from "@tauri-apps/plugin-sql";
import { isBoolean } from "es-toolkit";
import { Kysely, sql } from "kysely";
import { TauriSqliteDialect } from "kysely-dialect-tauri";
import { SerializePlugin } from "kysely-plugin-serialize";
import type { DatabaseSchema } from "@/types/database";
import { getSaveDatabasePath } from "@/utils/path";

let db: Kysely<DatabaseSchema> | null = null;

export const getDatabase = async () => {
  if (db) return db;

  const path = await getSaveDatabasePath();

  db = new Kysely<DatabaseSchema>({
    dialect: new TauriSqliteDialect({
      database: (prefix) => Database.load(prefix + path),
    }),
    plugins: [
      new SerializePlugin({
        deserializer: (value) => value,
        serializer: (value) => {
          if (isBoolean(value)) {
            return Number(value);
          }

          return value;
        },
      }),
    ],
  });

  // 启用 WAL 模式：读写并发，搜索时不阻塞剪贴板写入
  await sql`PRAGMA journal_mode = WAL`.execute(db);

  await db.schema
    .createTable("history")
    .ifNotExists()
    .addColumn("id", "text", (col) => col.primaryKey())
    .addColumn("type", "text")
    .addColumn("group", "text")
    .addColumn("value", "text")
    .addColumn("search", "text")
    .addColumn("count", "integer")
    .addColumn("width", "integer")
    .addColumn("height", "integer")
    .addColumn("favorite", "integer", (col) => col.defaultTo(0))
    .addColumn("createTime", "text")
    .addColumn("note", "text")
    .addColumn("subtype", "text")
    .execute();

  // 索引：大库下提升排序/筛选性能（尤其是 ORDER BY createTime DESC LIMIT）
  await db.schema
    .createIndex("idx_history_createTime")
    .ifNotExists()
    .on("history")
    .column("createTime")
    .execute();

  await db.schema
    .createIndex("idx_history_group_createTime")
    .ifNotExists()
    .on("history")
    .columns(["group", "createTime"])
    .execute();

  await db.schema
    .createIndex("idx_history_favorite_createTime")
    .ifNotExists()
    .on("history")
    .columns(["favorite", "createTime"])
    .execute();

  // FTS5 全文索引：用 trigram 分词器加速 LIKE '%keyword%' 搜索
  await sql`
    CREATE VIRTUAL TABLE IF NOT EXISTS history_fts USING fts5(
      search, note,
      content='history', content_rowid='rowid',
      tokenize='trigram', detail='none'
    )
  `.execute(db);

  // 触发器：history 表增删改时自动同步 FTS 索引
  await sql`
    CREATE TRIGGER IF NOT EXISTS history_fts_ai AFTER INSERT ON history BEGIN
      INSERT INTO history_fts(rowid, search, note) VALUES (NEW.rowid, NEW.search, NEW.note);
    END
  `.execute(db);

  await sql`
    CREATE TRIGGER IF NOT EXISTS history_fts_ad AFTER DELETE ON history BEGIN
      INSERT INTO history_fts(history_fts, rowid, search, note) VALUES ('delete', OLD.rowid, OLD.search, OLD.note);
    END
  `.execute(db);

  await sql`
    CREATE TRIGGER IF NOT EXISTS history_fts_au AFTER UPDATE ON history BEGIN
      INSERT INTO history_fts(history_fts, rowid, search, note) VALUES ('delete', OLD.rowid, OLD.search, OLD.note);
      INSERT INTO history_fts(rowid, search, note) VALUES (NEW.rowid, NEW.search, NEW.note);
    END
  `.execute(db);

  // 首次启动时，如果 FTS 表为空则灌入全量数据
  const ftsCount = await sql<{ cnt: number }>`
    SELECT count(*) AS cnt FROM history_fts
  `.execute(db);

  if (ftsCount.rows[0]?.cnt === 0) {
    const historyCount = await sql<{ cnt: number }>`
      SELECT count(*) AS cnt FROM history
    `.execute(db);

    if (historyCount.rows[0]?.cnt > 0) {
      await sql`
        INSERT INTO history_fts(rowid, search, note)
        SELECT rowid, search, note FROM history
      `.execute(db);
    }
  }

  return db;
};

export const destroyDatabase = async () => {
  const db = await getDatabase();

  return db.destroy();
};
