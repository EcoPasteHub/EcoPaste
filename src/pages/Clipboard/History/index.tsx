import type { HistoryItem } from "@/types/database";
import { appWindow } from "@tauri-apps/api/window";
import {
	onSomethingUpdate,
	readFilesURIs,
	readHtml,
	readImageBase64,
	readRtf,
	readText,
	startListening,
} from "tauri-plugin-clipboard-api";

const ClipboardWindow = () => {
	const [, setHistoryList] = useState<HistoryItem[]>();

	useMount(async () => {
		frostedWindow();

		getData();

		startListening();

		onSomethingUpdate(async (updateTypes) => {
			if (updateTypes.files) {
				await insertSQL("history", {
					type: "files",
					content: JSON.stringify(await readFilesURIs()),
				});
			} else if (updateTypes.image) {
				await insertSQL("history", {
					type: "image",
					content: `data:image/png;base64, ${await readImageBase64()}`,
				});
			} else if (updateTypes.html) {
				await insertSQL("history", {
					type: "html",
					content: await readHtml(),
				});
			} else if (updateTypes.rtf) {
				await insertSQL("history", {
					type: "rtf",
					content: await readRtf(),
				});
			} else if (updateTypes.text) {
				await insertSQL("history", {
					type: "text",
					content: await readText(),
				});
			}

			getData();
		});
	});

	const getData = async () => {
		const list = await selectSQL<HistoryItem[]>("history");

		setHistoryList(list);
	};

	return (
		<div
			className="h-screen rounded-8 p-16"
			onMouseDown={() => appWindow.startDragging()}
		>
			1
		</div>
	);
};

export default ClipboardWindow;
