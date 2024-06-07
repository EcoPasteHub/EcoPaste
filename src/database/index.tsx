import type { TableName, TablePayload } from "@/types/database";
import { getName } from "@tauri-apps/api/app";
import { appConfigDir } from "@tauri-apps/api/path";
import { isBoolean, map, snakeCase } from "lodash-es";
import Database from "tauri-plugin-sql-api";

let db: Database;

export const initDatabase = async () => {
	const appName = await getName();
	const configDir = await appConfigDir();
	const extensionName = isDev() ? "dev.db" : "db";

	db = await Database.load(`sqlite:${configDir}${appName}.${extensionName}`);

	await executeSQL(
		`
        CREATE TABLE IF NOT EXISTS history (id INTEGER PRIMARY KEY AUTOINCREMENT, type TEXT, content TEXT, create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP, is_favorite INTEGER DEFAULT 0);
        `,
	);
};

/**
 * 执行 sql 语句
 * @param sql sql 语句
 */
export const executeSQL = async (query: string, values?: unknown[]) => {
	if (!db) {
		await initDatabase();
	}

	if (query.startsWith("SELECT")) {
		return await db.select(query, values);
	}

	await db.execute(query, values);
};

/**
 * 查找的 sql 语句
 * @param tableName 表名称
 * @returns
 */
export const selectSQL = async (
	tableName: TableName,
	payload: TablePayload = {},
) => {
	const { keys, values } = handlePayload(payload);

	const clause = map(keys, (key) => `${key} LIKE ?`).join(" AND ");

	const whereClause = clause ? `WHERE ${clause}` : "";

	return await executeSQL(
		`SELECT * FROM ${tableName} ${whereClause} ORDER BY id DESC;`,
		map(values, (item) => `%${item}%`),
	);
};

/**
 * 添加的 sql 语句
 * @param tableName 表名称
 * @param payload 添加的数据
 */
export const insertSQL = async (
	tableName: TableName,
	payload: TablePayload,
) => {
	const { keys, values, refs } = handlePayload(payload);

	await executeSQL(
		`INSERT INTO ${tableName} (${keys}) VALUES (${refs});`,
		values,
	);
};

/**
 * 更新的 sql 语句
 * @param tableName 表名称
 * @param payload 修改的数据
 */
export const updateSQL = async (
	tableName: TableName,
	payload: TablePayload,
) => {
	const { id, ...rest } = payload;

	const { keys, values } = handlePayload(rest);

	const setClause = map(keys, (item) => `${item} = ?`);

	await executeSQL(
		`UPDATE ${tableName} SET ${setClause} WHERE id = ?;`,
		values.concat(id),
	);
};

/**
 * 删除的 sql 语句
 * @param tableName 表名称
 * @param id 删除数据的 id
 */
export const deleteSQL = async (tableName: TableName, id?: number) => {
	if (id) {
		await executeSQL(`DELETE FROM ${tableName} WHERE id = ?;`, [id]);
	} else {
		await executeSQL(`DELETE FROM ${tableName};`);
	}
};

/**
 * 处理参数
 * @param payload 数据
 */
const handlePayload = (payload: TablePayload) => {
	const keys = map(Object.keys(payload), snakeCase);
	const refs = map(keys, () => "?");
	const values = map(Object.values(payload), (item) => {
		return isBoolean(item) ? Number(item) : item;
	});

	return {
		keys,
		refs,
		values,
	};
};
