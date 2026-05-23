import { cloneDeep } from "es-toolkit";
import { isEmpty, remove } from "es-toolkit/compat";
import { nanoid } from "nanoid";
import { useEffect, useRef } from "react";
import {
	type ClipboardChangeOptions,
	onClipboardChange,
	startListening,
} from "tauri-plugin-clipboard-x-api";
import { fullName } from "tauri-plugin-fs-pro-api";
import {
	insertHistory,
	selectHistory,
	updateHistory,
} from "@/database/history";
import type { State } from "@/pages/Main";
import {
	getClipboardTextSubtype,
	shouldIgnoreClipboardTextHistory,
} from "@/plugins/clipboard";
import { clipboardStore } from "@/stores/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";
import { formatDate } from "@/utils/dayjs";

const RECENT_CLIPBOARD_EVENT_MS = 800;
const recentClipboardEvents = new Map<string, number>();

const markClipboardEvent = (key: string) => {
	const now = Date.now();

	for (const [entry, timestamp] of recentClipboardEvents.entries()) {
		if (now - timestamp > RECENT_CLIPBOARD_EVENT_MS) {
			recentClipboardEvents.delete(entry);
		}
	}

	const lastTime = recentClipboardEvents.get(key);

	if (lastTime && now - lastTime < RECENT_CLIPBOARD_EVENT_MS) {
		return true;
	}

	recentClipboardEvents.set(key, now);

	return false;
};

export const useClipboard = (
	state: State,
	options?: ClipboardChangeOptions,
) => {
	const optionsRef = useRef(options);

	optionsRef.current = options;

	useEffect(() => {
		let active = true;
		let unlisten: undefined | (() => void);

		const setup = async () => {
			await startListening();

			const dispose = await onClipboardChange(
				async (result) => {
					if (!active) return;

					const { files, image, html, rtf, text } = result;

					if (isEmpty(result) || Object.values(result).every(isEmpty)) return;
					if (text?.value && shouldIgnoreClipboardTextHistory(text.value))
						return;

					const { copyPlain } = clipboardStore.content;

					const data = {
						createTime: formatDate(),
						favorite: false,
						group: "text",
						id: nanoid(),
						search: text?.value,
					} as DatabaseSchemaHistory;

					if (files) {
						Object.assign(data, files, {
							group: "files",
							search: files.value.join(" "),
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
					} else if (image) {
						Object.assign(data, image, {
							group: "image",
						});
					}

					const sqlData = cloneDeep(data);

					const { type, value, group, createTime } = data;

					if (type === "image") {
						sqlData.value = await fullName(value);
					}

					if (type === "files") {
						sqlData.value = JSON.stringify(value);
					}

					const eventKey = `${sqlData.type}:${sqlData.value}`;

					if (markClipboardEvent(eventKey)) return;

					const [matched] = await selectHistory((qb) => {
						const { type, value } = sqlData;

						return qb.where("type", "=", type).where("value", "=", value);
					});

					const visible = state.group === "all" || state.group === group;

					if (matched) {
						if (!clipboardStore.content.autoSort) return;

						const { id } = matched;

						if (visible) {
							remove(state.list, { id });

							state.list.unshift({ ...data, id });
						}

						return updateHistory(id, { createTime });
					}

					if (visible) {
						state.list.unshift(data);
					}

					await insertHistory(sqlData);
				},
				{
					beforeRead() {
						optionsRef.current?.beforeRead?.();
					},
				},
			);

			if (!active) {
				dispose();
				return;
			}

			unlisten = dispose;
		};

		void setup();

		return () => {
			active = false;
			unlisten?.();
		};
	}, [state]);
};
