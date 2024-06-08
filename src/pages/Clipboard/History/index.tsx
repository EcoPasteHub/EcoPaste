import copyAudio from "@/assets/audio/copy.mp3";
import type { HistoryItem, TablePayload } from "@/types/database";
import { isEqual } from "lodash-es";
import { createContext } from "react";
import {
	onSomethingUpdate,
	readFilesURIs,
	readHtml,
	readImageBase64,
	readRtf,
	readText,
	startListening,
} from "tauri-plugin-clipboard-api";
import { useSnapshot } from "valtio";
import Header from "./components/Header";
import Popup from "./components/Popup";

interface State extends HistoryItem {
	historyList: HistoryItem[];
	previousPayload?: TablePayload;
}

interface HistoryContextValue {
	state: State;
	getHistoryList?: (payload?: HistoryItem) => void;
}

export const HistoryContext = createContext<HistoryContextValue>({
	state: {
		historyList: [],
	},
});

const ClipboardHistory = () => {
	const { wakeUpKey } = useSnapshot(clipboardStore);

	const audioRef = useRef<HTMLAudioElement>(null);

	const state = useReactive<State>({
		historyList: [],
	});

	useMount(async () => {
		frostedWindow();

		startListening();

		onSomethingUpdate(async (updateTypes) => {
			if (clipboardStore.enableAudio) {
				audioRef.current?.play();
			}

			let payload: TablePayload = {};

			if (updateTypes.files) {
				payload = {
					type: "files",
					group: "files",
					content: JSON.stringify(await readFilesURIs()),
				};
			} else if (updateTypes.image) {
				payload = {
					type: "image",
					group: "image",
					content: `data:image/png;base64, ${await readImageBase64()}`,
				};
			} else if (updateTypes.html) {
				payload = {
					type: "html",
					group: "text",
					content: await readHtml(),
				};
			} else if (updateTypes.rtf) {
				payload = {
					type: "rtf",
					group: "text",
					content: await readRtf(),
				};
			} else if (updateTypes.text) {
				payload = {
					type: "text",
					group: "text",
					content: await readText(),
				};
			}

			if (isEqual(payload, state.previousPayload)) return;

			await insertSQL("history", payload);

			state.previousPayload = payload;

			getHistoryList();
		});
	});

	useRegister(toggleWindowVisible, [wakeUpKey]);

	useEffect(() => {
		getHistoryList?.();
	}, [state.content, state.group, state.isFavorite]);

	const getHistoryList = async () => {
		const { content, group, isFavorite } = state;

		state.historyList = await selectSQL<HistoryItem[]>("history", {
			content,
			group,
			isFavorite,
		});
	};

	return (
		<div data-tauri-drag-region className="h-screen rounded-8 p-12">
			<audio ref={audioRef} src={copyAudio} />

			<HistoryContext.Provider
				value={{
					state,
					getHistoryList,
				}}
			>
				<Header />

				<Popup />
			</HistoryContext.Provider>
		</div>
	);
};

export default ClipboardHistory;
