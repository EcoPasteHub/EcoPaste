import copyAudio from "@/assets/audio/copy.mp3";
import type { HistoryGroup, HistoryItem } from "@/types/database";
import { listen } from "@tauri-apps/api/event";
import {
	PhysicalPosition,
	appWindow,
	currentMonitor,
} from "@tauri-apps/api/window";
import { isArray } from "arcdash";
import clsx from "clsx";
import { createContext } from "react";
import { useSnapshot } from "valtio";
import Header from "./components/Header";
import List from "./components/List";

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

		onClipboardUpdate(async (payload) => {
			if (clipboardStore.enableAudio) {
				audioRef.current?.play();
			}

			let { type, value } = payload;

			value = isArray(value) ? JSON.stringify(value) : value;

			const [selectItem] = await selectSQL<HistoryItem[]>("history", {
				...payload,
				value,
			});

			if (selectItem) {
				await updateSQL("history", {
					id: selectItem.id,
					createTime: dayjs().format("YYYY-MM-DD HH:mm:ss"),
				});
			} else {
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
					...payload,
					value,
					group,
				});
			}

			getHistoryList();
		});

		listen(LISTEN_KEY.CLEAR_HISTORY, async () => {
			await deleteSQL("history");

			getHistoryList();
		});
	});

	useRegister(async () => {
		const focused = await appWindow.isFocused();

		if (!focused) {
			const { windowPosition } = clipboardStore;

			if (windowPosition === "follow") {
				const monitor = await currentMonitor();

				if (!monitor) return;

				const { width, height } = await appWindow.innerSize();
				let [x, y] = await getMouseCoords();

				const {
					scaleFactor,
					size: { width: screenWidth, height: screenHeight },
				} = monitor;

				const factor = (await isWin()) ? 1 : scaleFactor;

				x = Math.min(x * factor, screenWidth - width);
				y = Math.min(y * factor, screenHeight - height);

				appWindow.setPosition(new PhysicalPosition(x, y));
			} else if (windowPosition === "center") {
				appWindow.center();
			}
		}

		toggleWindowVisible();
	}, [wakeUpKey]);

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

				<List />
			</HistoryContext.Provider>
		</div>
	);
};

export default ClipboardHistory;
