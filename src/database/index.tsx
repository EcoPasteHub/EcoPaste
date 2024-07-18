import type {
	HistoryItem,
	SelectPayload,
	TableName,
	TablePayload,
} from "@/types/database";
import { getName } from "@tauri-apps/api/app";
import { removeFile } from "@tauri-apps/api/fs";
import { appDataDir } from "@tauri-apps/api/path";
import { isBoolean, isNil, map, omitBy } from "lodash-es";
import Database from "tauri-plugin-sql-api";

let db: Database;

/**
 * 处理参数
 * @param payload 数据
 */
const handlePayload = (payload: TablePayload) => {
	const omitPayload = omitBy(payload, isNil);

	const keys = map(Object.keys(omitPayload), (key) => `[${key}]`);
	const refs = map(keys, () => "?");
	const values = map(Object.values(omitPayload), (item) => {
		return isBoolean(item) ? Number(item) : item;
	});

	return {
		keys,
		refs,
		values,
	};
};

/**
 * 初始化数据库
 */
export const initDatabase = async () => {
	const appName = await getName();
	const dataDir = await appDataDir();
	const extensionName = isDev() ? "dev.db" : "db";

	db = await Database.load(`sqlite:${dataDir}${appName}.${extensionName}`);

	await executeSQL(
		`
        CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT,
			[group] TEXT,
			value TEXT,
     		search TEXT,
			width INTEGER,
			height INTEGER,
			size INTEGER,
			createTime TIMESTAMP DEFAULT (DATETIME('now', 'localtime')),
			isCollected INTEGER DEFAULT 0
		);
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
export const selectSQL = async <List,>(
	tableName: TableName,
	payload: SelectPayload = {},
) => {
	const { exact, ...rest } = payload;

	const { keys, values } = handlePayload(rest);

	const connector = exact ? "=" : "LIKE";

	const clause = map(keys, (key) => `${key} ${connector} ?`).join(" AND ");

	const whereClause = clause ? `WHERE ${clause}` : "";

	const bindValues = exact ? values : values.map((item) => `%${item}%`);

	const list = await executeSQL(
		`SELECT * FROM ${tableName} ${whereClause} ORDER BY createTime DESC;`,
		bindValues,
	);

	return (list ?? []) as List;
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
		values.concat(id!),
	);
};

/**
 * 删除的 sql 语句
 * @param tableName 表名称
 * @param id 删除数据的 id
 */
export const deleteSQL = async (tableName: TableName, id?: number) => {
	const list = await selectSQL<HistoryItem[]>("history", { id });

	const deleteImageFile = (item: HistoryItem) => {
		const { type, value = "" } = item;

		if (type !== "image") return;

		removeFile(clipboardStore.saveImageDir + value);
	};

	if (id) {
		await executeSQL(`DELETE FROM ${tableName} WHERE id = ?;`, [id]);

		deleteImageFile(list[0]);
	} else {
		await executeSQL(`DELETE FROM ${tableName};`);

		for (const item of list) {
			deleteImageFile(item);
		}
	}
};

/**
 * 关闭数据库连接池
 */
export const closeDatabase = async () => {
	await db.close();
};
