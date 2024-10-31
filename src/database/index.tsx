import type { ClipboardItem, TableName, TablePayload } from "@/types/database";
import { getName } from "@tauri-apps/api/app";
import Database from "@tauri-apps/plugin-sql";
import { entries, find, isBoolean, isNil, map, omitBy, some } from "lodash-es";

let db: Database | null;

/**
 * 初始化数据库
 */
export const initDatabase = async () => {
	const appName = await getName();
	const extname = isDev() ? "dev.db" : "db";
	const path = joinPath(getSaveDataDir(), `${appName}.${extname}`);

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
			note TEXT,
			subtype TEXT
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

	const fields = await getFields("history");

	// `isCollected` 更名 `favorite`
	await renameField("history", "isCollected", "favorite");

	// `size` 更名 `count`
	await renameField("history", "size", "count");

	if (!some(fields, { name: "note" })) {
		// 新增 `remark`
		await addField("history", "remark", "TEXT");

		// `remark` 更名 `note`
		await renameField("history", "remark", "note");
	}

	// 将 `id` 从 INTEGER 转为 TEXT 类型
	if (find(fields, { name: "id" })?.type === "INTEGER") {
		const tableName = "temp_history";

		await executeSQL(createHistoryQuery(tableName));

		await executeSQL(
			`INSERT INTO ${tableName} (id, type, [group], value, search, count, width, height, favorite, createTime, note, subtype) SELECT CAST(id AS TEXT), type, [group], value, search, count, width, height, favorite, createTime, note, subtype FROM history;`,
		);

		await executeSQL("DROP TABLE history;");

		await executeSQL(`ALTER TABLE ${tableName} RENAME TO history;`);
	}

	// 新增 `subtype`
	if (!some(fields, { name: "subtype" })) {
		await addField("history", "subtype", "TEXT");

		const list = await selectSQL<ClipboardItem[]>("history");

		for await (const item of list) {
			const { id, type } = item;

			if (type !== "text") return;

			const subtype = await getClipboardSubtype(item);

			if (!subtype) return;

			await updateSQL("history", { id, subtype });
		}
	}
};

/**
 * 处理参数
 * @param payload 数据
 */
const handlePayload = (payload: TablePayload) => {
	const omitPayload = omitBy(payload, isNil);

	const keys = [];
	const values = [];

	for (const [key, value] of entries(omitPayload)) {
		keys.push(key === "group" ? "[group]" : key);

		const nextValue = isBoolean(value) ? Number(value) : value;

		values.push(nextValue);
	}

	return {
		keys,
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
	payload: TablePayload = {},
) => {
	const { keys, values } = handlePayload(payload);

	const clause = map(keys, (key, index) => {
		if (key === "search") {
			const value = `%${payload.search}%`;

			values[index] = value;
			values.splice(index + 1, 0, value);

			return "(search LIKE ? OR note LIKE ?)";
		}

		return `${key} = ?`;
	}).join(" AND ");

	const whereClause = clause ? `WHERE ${clause}` : "";

	const list = await executeSQL(
		`SELECT * FROM ${tableName} ${whereClause} ORDER BY createTime DESC;`,
		values,
	);

	return (list ?? []) as List;
};

/**
 * 添加的 sql 语句
 * @param tableName 表名称
 * @param payload 添加的数据
 */
export const insertSQL = (tableName: TableName, payload: TablePayload) => {
	const { keys, values } = handlePayload(payload);

	const refs = map(values, () => "?");

	return executeSQL(
		`INSERT INTO ${tableName} (${keys}) VALUES (${refs});`,
		values,
	);
};

/**
 * 更新的 sql 语句
 * @param tableName 表名称
 * @param payload 修改的数据
 */
export const updateSQL = (tableName: TableName, payload: TablePayload) => {
	const { id, ...rest } = payload;

	const { keys, values } = handlePayload(rest);

	if (keys.length === 0) return;

	const setClause = map(keys, (item) => `${item} = ?`);

	return executeSQL(
		`UPDATE ${tableName} SET ${setClause} WHERE id = ?;`,
		values.concat(id!),
	);
};

/**
 * 删除的 sql 语句
 * @param tableName 表名称
 * @param id 删除数据的 id
 */
export const deleteSQL = async (tableName: TableName, item: ClipboardItem) => {
	const { id, type, value } = item;

	await executeSQL(`DELETE FROM ${tableName} WHERE id = ?;`, [id]);

	if (type !== "image") return;

	return removeFile(getSaveImagePath(value));
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
