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

	await executeSQL(
		`
        CREATE TABLE IF NOT EXISTS history (
			id TEXT PRIMARY KEY,
			type TEXT,
			[group] TEXT,
			value TEXT,
			search TEXT,
			count INTEGER,
			width INTEGER,
			height INTEGER,
			favorite INTEGER DEFAULT 0,
			createTime TEXT
		);
        `,
	);

	// 以下代码未来版本会删除！！！
	// 获取 history 表字段
	const fields: any = await executeSQL("PRAGMA table_info(history)");

	// `isCollected` 更名 `favorite`
	if (some(fields, { name: "isCollected" })) {
		await executeSQL(
			"ALTER TABLE history RENAME COLUMN isCollected TO favorite;",
		);
	}

	// `size` 更名 `count`
	if (some(fields, { name: "size" })) {
		await executeSQL("ALTER TABLE history RENAME COLUMN size TO count;");
	}

	// 将 type 为富文本的简化为 rtf
	await executeSQL("UPDATE history SET type = ? WHERE type = ?;", [
		"rtf",
		"rich-text",
	]);

	// 将 id 类型从 INTEGER 转为 TEXT
	if (find(fields, { name: "id" })?.type === "INTEGER") {
		await executeSQL(
			`
			CREATE TABLE IF NOT EXISTS temp_history (
				id TEXT PRIMARY KEY,
				type TEXT,
				[group] TEXT,
				value TEXT,
				search TEXT,
				count INTEGER,
				width INTEGER,
				height INTEGER,
				favorite INTEGER DEFAULT 0,
				createTime TEXT
			);
			`,
		);

		await executeSQL(
			"INSERT INTO temp_history (id, type, [group], value, search, count, width, height, favorite, createTime) SELECT CAST(id AS TEXT), type, [group], value, search, count, width, height, favorite, createTime FROM history;",
		);

		await executeSQL("DROP TABLE history;");

		await executeSQL("ALTER TABLE temp_history RENAME TO history;");
	}
	// 以上代码未来版本会删除！！！
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
