import type { HistoryItem } from "@/types/database";
import {
	onSomethingUpdate,
	readFilesURIs,
	readHtml,
	readImageBase64,
	readRtf,
	readText,
	startListening,
} from "tauri-plugin-clipboard-api";

type Cb = (newValue: HistoryItem, oldValue?: HistoryItem) => void;

export const useReadClipboard = (cb: Cb) => {
	const oldValue = useRef<HistoryItem>();

	useMount(async () => {
		await startListening();

		onSomethingUpdate(async (updateTypes) => {
			let value: HistoryItem = {};

			if (updateTypes.files) {
				const filesURIs = await readFilesURIs();

				value = {
					type: "files",
					group: "files",
					content: JSON.stringify(filesURIs.map(decodeURIComponent)),
				};
			} else if (updateTypes.image) {
				value = {
					type: "image",
					group: "image",
					content: await readImageBase64(),
				};
			} else if (updateTypes.html) {
				value = {
					type: "html",
					group: "text",
					content: await readHtml(),
				};
			} else if (updateTypes.rtf) {
				value = {
					type: "rtf",
					group: "text",
					content: await readRtf(),
				};
			} else if (updateTypes.text) {
				value = {
					type: "text",
					group: "text",
					content: await readText(),
				};
			}

			cb(value, oldValue.current);

			oldValue.current = value;
		});
	});
};
