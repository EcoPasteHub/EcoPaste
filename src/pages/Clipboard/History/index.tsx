import copyAudio from "@/assets/audio/copy.mp3";
import type { HistoryGroup, HistoryItem } from "@/types/database";
import { listen } from "@tauri-apps/api/event";
import clsx from "clsx";
import { isArray, isEqual } from "lodash-es";
import { createContext } from "react";
import { useSnapshot } from "valtio";
import Header from "./components/Header";
import Popup from "./components/Popup";

interface State extends HistoryItem {
	historyList: HistoryItem[];
}

const INITIAL_STATE: State = {
	historyList: [],
};

interface HistoryContextValue {
	state: State;
	getHistoryList?: (payload?: HistoryItem) => void;
}

export const HistoryContext = createContext<HistoryContextValue>({
	state: INITIAL_STATE,
});

const ClipboardHistory = () => {
	const { wakeUpKey } = useSnapshot(clipboardStore);

	const audioRef = useRef<HTMLAudioElement>(null);

	const state = useReactive<State>(INITIAL_STATE);

	useMount(async () => {
		await initDatabase();

		startListen();

		onClipboardUpdate(async (newPayload, oldPayload) => {
			if (clipboardStore.enableAudio) {
				audioRef.current?.play();
			}

			if (isEqual(newPayload, oldPayload)) return;

			const { type, value } = newPayload;

			let group: HistoryGroup;

			switch (type) {
				case "files":
					group = "files";
					break;
				case "image":
					group = "image";
					break;
				default:
					group = "text";
					break;
			}

			await insertSQL("history", {
				...newPayload,
				group,
				value: isArray(value) ? JSON.stringify(value) : value,
			});

			getHistoryList();
		});

		listen(LISTEN_KEY.CLEAR_HISTORY, async () => {
			await deleteSQL("history");

			getHistoryList();
		});
	});

	useRegister(toggleWindowVisible, [wakeUpKey]);

	useEffect(() => {
		getHistoryList?.();
	}, [state.value, state.group, state.isCollected]);

	const getHistoryList = async () => {
		const { value, group, isCollected } = state;

		const list = await selectSQL<HistoryItem[]>("history", {
			value,
			group,
			isCollected,
		});

		state.historyList = list;

		if (!clipboardStore.capacity) return;

		for (const item of list) {
			const { id, createTime } = item;

			if (dayjs().diff(createTime, "days") >= clipboardStore.capacity) {
				deleteSQL("history", id);
			}
		}
	};

	return (
		<div
			data-tauri-drag-region
			className={clsx("h-screen rounded-10 bg-1 p-12")}
		>
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
