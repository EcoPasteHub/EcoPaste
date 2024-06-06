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
	useMount(() => {
		frostedWindow();

		startListening();

		onSomethingUpdate(async (updateTypes) => {
			if (updateTypes.files) {
				return insertSQL("history", {
					type: "files",
					content: JSON.stringify(await readFilesURIs()),
				});
			}

			if (updateTypes.image) {
				return insertSQL("history", {
					type: "image",
					content: `data:image/png;base64, ${await readImageBase64()}`,
				});
			}

			if (updateTypes.html) {
				return insertSQL("history", {
					type: "html",
					content: await readHtml(),
				});
			}

			if (updateTypes.rtf) {
				return insertSQL("history", {
					type: "rtf",
					content: await readRtf(),
				});
			}

			if (updateTypes.text) {
				insertSQL("history", {
					type: "text",
					content: await readText(),
				});
			}
		});
	});

	return (
		<div className="h-screen" onMouseDown={() => appWindow.startDragging()}>
			ClipboardWindow
		</div>
	);
};

export default ClipboardWindow;
