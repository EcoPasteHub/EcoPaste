import type {
	ClipboardItem,
	SelectPayload,
	TableName,
	TablePayload,
} from "@/types/database";
import { getName } from "@tauri-apps/api/app";
import { removeFile } from "@tauri-apps/api/fs";
import { find, isBoolean, isNil, map, omitBy, some } from "lodash-es";
import Database from "tauri-plugin-sql-api";

let db: Database | null;

/**
 * 初始化数据库
 */
export const initDatabase = async () => {
	const appName = await getName();
	const ext = isDev() ? "dev.db" : "db";
	const path = joinPath(getSaveDataDir(), `${appName}.${ext}`);

	db = await Database.load(`sqlite:${path}`);

	const createHistoryQuery = (tableName = "history") => {
		return `
        CREATE TABLE IF NOT EXISTS ${tableName} (
			id TEXT PRIMARY KEY,
			type TEXT,
			[group] TEXT,
			value TEXT,
			search TEXT,
			count INTEGER,
			width INTEGER,
			height INTEGER,
			favorite INTEGER DEFAULT 0,
			createTime TEXT,
			remark TEXT
		);
        `;
	};

	// 创建 `history` 表
	await executeSQL(createHistoryQuery());

	// 将 `type` 为 rich-text 的更换为 rtf
	await executeSQL("UPDATE history SET type = ? WHERE type = ?;", [
		"rtf",
		"rich-text",
	]);

	// `isCollected` 更名 `favorite`
	await renameField("history", "isCollected", "favorite");

	// `size` 更名 `count`
	await renameField("history", "size", "count");

	// 新增 `remark`
	await addField("history", "remark", "TEXT");

	// 将 `id` 从 INTEGER 转为 TEXT 类型
	const fields = await getFields("history");
	if (find(fields, { name: "id" })?.type === "INTEGER") {
		const tableName = "temp_history";

		await executeSQL(createHistoryQuery(tableName));

		await executeSQL(
			`INSERT INTO ${tableName} (id, type, [group], value, search, count, width, height, favorite, createTime, remark) SELECT CAST(id AS TEXT), type, [group], value, search, count, width, height, favorite, createTime, remark FROM history;`,
		);

		await executeSQL("DROP TABLE history;");

		await executeSQL(`ALTER TABLE ${tableName} RENAME TO history;`);
	}
};

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
 * 执行 sql 语句
 * @param sql sql 语句
 */
export const executeSQL = async (query: string, values?: unknown[]) => {
	if (!db) {
		await initDatabase();
	}

	if (query.startsWith("SELECT") || query.startsWith("PRAGMA")) {
		return await db!.select(query, values);
	}

	await db!.execute(query, values);
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
export const deleteSQL = async (tableName: TableName, id: string) => {
	const [item] = await selectSQL<ClipboardItem[]>("history", { id });

	await executeSQL(`DELETE FROM ${tableName} WHERE id = ?;`, [id]);

	const { type, value = "" } = item;

	if (type !== "image") return;

	removeFile(getSaveImagePath(value));
};

/**
 * 关闭数据库连接池
 */
export const closeDatabase = async () => {
	await db?.close();

	db = null;
};

/**
 * 获取全部字段
 * @param tableName 表名
 */
const getFields = async (tableName: TableName) => {
	const fields = await executeSQL(`PRAGMA table_info(${tableName})`);

	return fields as { name: string; type: string }[];
};

/**
 * 重命名字段
 * @param tableName 表名
 * @param field 字段名称
 * @param rename 重命名
 * @returns
 */
const renameField = async (
	tableName: TableName,
	field: string,
	rename: string,
) => {
	const fields = await getFields(tableName);

	if (some(fields, { name: rename })) return;

	return executeSQL(
		`ALTER TABLE ${tableName} RENAME COLUMN ${field} TO ${rename};`,
	);
};

/**
 * 新增字段
 * @param tableName 表名
 * @param field 字段
 * @param type 类型
 */
const addField = async (tableName: TableName, field: string, type: string) => {
	const fields = await getFields(tableName);

	if (some(fields, { name: field })) return;

	return executeSQL(`ALTER TABLE ${tableName} ADD COLUMN ${field} ${type};`);
};
