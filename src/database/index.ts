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

  return db;
};

export const destroyDatabase = async () => {
  const db = await getDatabase();

  return db.destroy();
};
