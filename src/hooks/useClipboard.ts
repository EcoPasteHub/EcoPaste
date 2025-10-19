import type { State } from "@/pages/Main";
import type { DatabaseSchemaHistory } from "@/types/database";
import { remove } from "es-toolkit/compat";
import { nanoid } from "nanoid";
import {
	onClipboardChange,
	startListening,
} from "tauri-plugin-clipboard-x-api";
import { fullName } from "tauri-plugin-fs-pro-api";

export const useClipboard = (state: State) => {
	useMount(async () => {
		await startListening();

		onClipboardChange(
			async (result) => {
				const { files, image, html, rtf, text } = result;
				const { copyPlain } = clipboardStore.content;

				const data = {
					id: nanoid(),
					createTime: formatDate(),
					favorite: false,
					group: "text",
					search: text?.value,
				} as DatabaseSchemaHistory;

				if (files) {
					Object.assign(data, files, {
						group: "files",
						search: files.value.join(" "),
					});
				} else if (image) {
					Object.assign(data, image, {
						group: "image",
					});
				} else if (html && !copyPlain) {
					Object.assign(data, html);
				} else if (rtf && !copyPlain) {
					Object.assign(data, rtf);
				} else if (text) {
					const subtype = await getClipboardTextSubtype(text.value);

					Object.assign(data, text, {
						subtype,
					});
				}

				const { type, value, group, createTime } = data;

				let dbValue = value;

				if (type === "image") {
					dbValue = await fullName(value);
				}

				if (type === "files") {
					dbValue = JSON.stringify(value);
				}

				const db = await getDatabase();
				const matched = await db
					.selectFrom("history")
					.select("id")
					.where("type", "=", type)
					.where("value", "=", dbValue)
					.executeTakeFirst();

				if (matched) {
					if (!clipboardStore.content.autoSort) return;

					const { id } = matched;

					const [targetItem] = remove(state.list, { id });

					state.list.unshift({ ...targetItem, createTime });

					return db
						.updateTable("history")
						.set("createTime", createTime)
						.where("id", "=", matched.id)
						.execute();
				}

				if (state.group === "all" || state.group === group) {
					state.list.unshift(data);
				}

				db.insertInto("history")
					.values({ ...data, value: dbValue })
					.execute();
			},
			{
				beforeRead: playAudio,
			},
		);
	});
};
