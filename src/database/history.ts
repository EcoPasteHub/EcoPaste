import { exists, remove } from "@tauri-apps/plugin-fs";
import type { AnyObject } from "antd/es/_util/type";
import { type SelectQueryBuilder, sql } from "kysely";
import { getDefaultSaveImagePath } from "tauri-plugin-clipboard-x-api";
import type { DatabaseSchema, DatabaseSchemaHistory } from "@/types/database";
import { join } from "@/utils/path";
import { getDatabase } from ".";

type QueryBuilder = SelectQueryBuilder<DatabaseSchema, "history", AnyObject>;

// 列表查询时 value 截断长度（只用于预览，避免几 MB 的文本拖慢渲染）
const LIST_VALUE_TRUNCATE = 500;

/**
 * 查询历史记录（列表用，value 字段截断以提升性能）
 */
export const selectHistory = async (
  fn?: (qb: QueryBuilder) => QueryBuilder,
) => {
  const db = await getDatabase();

  // 不用 selectAll()，而是显式选择字段，value 截断到 500 字符
  let qb = db
    .selectFrom("history")
    .select([
      "id",
      "type",
      "group",
      sql<string>`substr(value, 1, ${sql.lit(LIST_VALUE_TRUNCATE)})`.as(
        "value",
      ),
      "search",
      "count",
      "width",
      "height",
      "favorite",
      "createTime",
      "note",
      "subtype",
    ]) as QueryBuilder;

  if (fn) {
    qb = fn(qb);
  }

  return qb.execute() as Promise<DatabaseSchemaHistory[]>;
};

/**
 * 获取单条记录的完整 value（粘贴/导出/复制时用）
 */
export const getHistoryFullValue = async (
  id: string,
): Promise<string | undefined> => {
  const db = await getDatabase();

  const result = await db
    .selectFrom("history")
    .select("value")
    .where("id", "=", id)
    .executeTakeFirst();

  return result?.value as string | undefined;
};

export const insertHistory = async (data: DatabaseSchemaHistory) => {
  const db = await getDatabase();

  return db.insertInto("history").values(data).execute();
};

export const updateHistory = async (
  id: string,
  nextData: Partial<DatabaseSchemaHistory>,
) => {
  const db = await getDatabase();

  return db.updateTable("history").set(nextData).where("id", "=", id).execute();
};

export const deleteHistory = async (data: DatabaseSchemaHistory) => {
  const { id, type, value } = data;

  const db = await getDatabase();

  await db.deleteFrom("history").where("id", "=", id).execute();

  if (type !== "image") return;

  let path = value;

  const saveImagePath = await getDefaultSaveImagePath();

  if (!value.startsWith(saveImagePath)) {
    path = join(saveImagePath, value);
  }

  const existed = await exists(path);

  if (!existed) return;

  return remove(path);
};
