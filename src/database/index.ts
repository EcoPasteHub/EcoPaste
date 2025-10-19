import type { DatabaseSchema } from "@/types/database";
import Database from "@tauri-apps/plugin-sql";
import { isBoolean } from "es-toolkit";
import { Kysely } from "kysely";
import { TauriSqliteDialect } from "kysely-dialect-tauri";
import { SerializePlugin } from "kysely-plugin-serialize";

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
				serializer: (value) => {
					if (isBoolean(value)) {
						return Number(value);
					}

					return value;
				},
				deserializer: (value) => value,
			}),
		],
	});

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

	return db;
};
