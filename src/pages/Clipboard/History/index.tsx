import copyAudio from "@/assets/audio/copy.mp3";
import type { HistoryItem } from "@/types/database";
import { listen } from "@tauri-apps/api/event";
import clsx from "clsx";
import { isEqual } from "lodash-es";
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

		listen(LISTEN_KEY.CLEAR_HISTORY, async () => {
			await deleteSQL("history");

			getHistoryList();
		});
	});

	useListenClipboard(async (newValue, oldValue) => {
		if (clipboardStore.enableAudio) {
			audioRef.current?.play();
		}

		if (isEqual(newValue, oldValue)) return;

		await insertSQL("history", newValue);

		getHistoryList();
	});

	useRegister(toggleWindowVisible, [wakeUpKey]);

	useEffect(() => {
		getHistoryList?.();
	}, [state.content, state.group, state.isCollected]);

	const getHistoryList = async () => {
		const { content, group, isCollected } = state;

		const list = await selectSQL<HistoryItem[]>("history", {
			content,
			group,
			isCollected,
		});

		state.historyList = list;

		if (!clipboardStore.capacity) return;

		const maxMinute = clipboardStore.capacity * 24 * 60;

		for (const item of list) {
			const { id, createTime } = item;

			if (dayjs().diff(createTime, "minutes") > maxMinute) {
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
