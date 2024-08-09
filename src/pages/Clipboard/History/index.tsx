import type { PlayAudioRef } from "@/components/PlayAudio";
import PlayAudio from "@/components/PlayAudio";
import type { HistoryItem, TablePayload } from "@/types/database";
import { listen } from "@tauri-apps/api/event";
import {
	PhysicalPosition,
	appWindow,
	currentMonitor,
} from "@tauri-apps/api/window";
import { Flex } from "antd";
import { isEqual } from "arcdash";
import clsx from "clsx";
import { merge } from "lodash-es";
import { createContext } from "react";
import { useSnapshot } from "valtio";
import Header from "./components/Header";
import List from "./components/List";
import Search from "./components/Search";

interface State extends TablePayload {
	rounded: boolean;
	pin?: boolean;
	historyList: HistoryItem[];
	activeIndex: number;
	searching?: boolean;
	scrollToIndex?: (index: number) => void;
}

const INITIAL_STATE: State = {
	rounded: true,
	historyList: [],
	activeIndex: 0,
};

interface HistoryContextValue {
	state: State;
	getHistoryList?: (payload?: HistoryItem) => Promise<void>;
}

export const HistoryContext = createContext<HistoryContextValue>({
	state: INITIAL_STATE,
});

const ClipboardHistory = () => {
	const { env, shortcut } = useSnapshot(globalStore);
	const { search } = useSnapshot(clipboardStore);

	const audioRef = useRef<PlayAudioRef>(null);

	const state = useReactive<State>(INITIAL_STATE);

	useMount(async () => {
		appWindow.setTitle(env.appName!);

		createWindow("/preference");

		setWindowShadow();

		startListen();

		onClipboardUpdate(async (payload) => {
			if (clipboardStore.audio.copy) {
				audioRef.current?.play();
			}

			const [selectItem] = await selectSQL<HistoryItem[]>("history", {
				...payload,
				exact: true,
			});

			if (selectItem) {
				await updateSQL("history", {
					id: selectItem.id,
					createTime: dayjs().format("YYYY-MM-DD HH:mm:ss"),
				});
			} else {
				let group: HistoryItem["group"];

				switch (payload.type) {
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
					group,
				});
			}

			getHistoryList();
		});

		listen(LISTEN_KEY.CLEAR_HISTORY, async () => {
			await deleteSQL("history");

			getHistoryList();
		});

		listen(LISTEN_KEY.IMPORT_DATA, async () => {
			await initDatabase();

			getHistoryList();
		});

		listen<boolean>(LISTEN_KEY.TOGGLE_LISTENING, ({ payload }) => {
			if (payload) {
				startListen();
			} else {
				stopListen();
			}
		});

		listen(LISTEN_KEY.GLOBAL_STORE_CHANGED, ({ payload }) => {
			if (isEqual(globalStore, payload)) return;

			merge(globalStore, payload);
		});

		listen(LISTEN_KEY.CLIPBOARD_STORE_CHANGED, ({ payload }) => {
			if (isEqual(clipboardStore, payload)) return;

			merge(clipboardStore, payload);
		});
	});

	useRegister(async () => {
		const focused = await appWindow.isFocused();

		if (!focused) {
			const { window } = clipboardStore;

			if (window.position === "follow") {
				const monitor = await currentMonitor();

				if (!monitor) return;

				const { width, height } = await appWindow.innerSize();
				let [x, y] = await getMouseCoords();

				const {
					scaleFactor,
					size: { width: screenWidth, height: screenHeight },
				} = monitor;

				const factor = isWin() ? 1 : scaleFactor;

				x = Math.min(x * factor, screenWidth - width);
				y = Math.min(y * factor, screenHeight - height);

				appWindow.setPosition(new PhysicalPosition(x, y));
			} else if (window.position === "center") {
				appWindow.center();
			}
		}

		toggleWindowVisible();
	}, [shortcut.clipboard]);

	const getHistoryList = async () => {
		const { search, group, isCollected } = state;

		const list = await selectSQL<HistoryItem[]>("history", {
			search,
			group,
			isCollected,
		});

		state.historyList = list;

		if (!clipboardStore.history.duration) return;

		for (const item of list) {
			const { id, createTime } = item;

			if (dayjs().diff(createTime, "days") >= clipboardStore.history.duration) {
				if (item.isCollected) continue;

				deleteSQL("history", id);
			}
		}
	};

	return (
		<HistoryContext.Provider
			value={{
				state,
				getHistoryList,
			}}
		>
			<Flex
				data-tauri-drag-region
				vertical
				gap={12}
				className={clsx("h-screen bg-1 py-12", {
					"rounded-10": !isWin(),
					"flex-col-reverse": search.position === "bottom",
				})}
			>
				<PlayAudio hiddenIcon ref={audioRef} />

				<Search />

				<Flex
					data-tauri-drag-region
					vertical
					gap={12}
					className="flex-1 overflow-hidden"
				>
					<Header />

					<List />
				</Flex>
			</Flex>
		</HistoryContext.Provider>
	);
};

export default ClipboardHistory;
