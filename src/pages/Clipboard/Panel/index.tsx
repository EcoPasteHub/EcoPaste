import type { AudioRef } from "@/components/Audio";
import Audio from "@/components/Audio";
import type { ClipboardItem, TablePayload } from "@/types/database";
import { listen } from "@tauri-apps/api/event";
import type { EventEmitter } from "ahooks/lib/useEventEmitter";
import { isEqual, merge } from "lodash-es";
import { createContext } from "react";
import { useSnapshot } from "valtio";
import Dock from "./components/Dock";
import Float from "./components/Float";

interface State extends TablePayload {
	pin?: boolean;
	activeId: number;
	data: {
		list: ClipboardItem[];
		total: number;
		page: number;
		size: number;
		loading: boolean;
	};
	$eventBus?: EventEmitter<string>;
	scrollToIndex?: (index: number) => void;
}

const INITIAL_STATE: State = {
	activeId: 0,
	data: {
		list: [],
		total: 0,
		page: 1,
		size: 20,
		loading: false,
	},
};

interface ClipboardPanelContextValue {
	state: State;
	getClipboardList?: (payload?: ClipboardItem) => Promise<void>;
}

export const ClipboardPanelContext = createContext<ClipboardPanelContextValue>({
	state: INITIAL_STATE,
});

const ClipboardPanel = () => {
	const { shortcut } = useSnapshot(globalStore);
	const { window } = useSnapshot(clipboardStore);
	const state = useReactive<State>(INITIAL_STATE);
	const audioRef = useRef<AudioRef>(null);
	const $eventBus = useEventEmitter<string>();

	useMount(async () => {
		state.$eventBus = $eventBus;

		startListen();

		onClipboardUpdate(async (payload) => {
			if (clipboardStore.audio.copy) {
				audioRef.current?.play();
			}

			const [selectItem] = await selectSQL<ClipboardItem[]>("history", {
				...payload,
				exact: true,
			});

			if (selectItem) {
				await updateSQL("history", {
					id: selectItem.id,
					createTime: dayjs().format("YYYY-MM-DD HH:mm:ss"),
				});
			} else {
				let group: ClipboardItem["group"];

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

			getClipboardList();
		});

		listen(LISTEN_KEY.CLEAR_HISTORY, async () => {
			await deleteSQL("history");

			getClipboardList();
		});

		listen(LISTEN_KEY.IMPORT_DATA, async () => {
			await initDatabase();

			getClipboardList();
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

		listen(LISTEN_KEY.TOGGLE_MAIN_WINDOW_VISIBLE, toggleWindowVisible);
	});

	useFocus({
		onBlur() {
			if (state.pin) return;

			hideWindow();
		},
	});

	useRegister(toggleWindowVisible, [shortcut.clipboard]);

	const getClipboardList = async () => {
		const { search, group, isCollected } = state;

		const list = await selectSQL<ClipboardItem[]>("history", {
			search,
			group,
			isCollected,
		});

		state.data.list = list;

		if (state.data.page === 1) {
			state.activeId = list[0]?.id;
		}

		const { duration, unit } = clipboardStore.history;

		if (duration === 0) return;

		for (const item of list) {
			const { id, createTime } = item;

			if (dayjs().diff(createTime, "days") >= duration * unit) {
				if (item.isCollected) continue;

				deleteSQL("history", id);
			}
		}
	};

	return (
		<>
			{!isLinux() && <Audio hiddenIcon ref={audioRef} />}

			<ClipboardPanelContext.Provider
				value={{
					state,
					getClipboardList,
				}}
			>
				{window.style === "float" ? <Float /> : <Dock />}
			</ClipboardPanelContext.Provider>
		</>
	);
};

export default ClipboardPanel;
