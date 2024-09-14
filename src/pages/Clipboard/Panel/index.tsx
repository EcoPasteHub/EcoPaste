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

		// 开启监听
		startListen();

		// 监听剪切板更新
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
				await insertSQL("history", payload);
			}

			getClipboardList();
		});

		// 监听清空历史记录
		listen(LISTEN_KEY.CLEAR_HISTORY, async () => {
			await deleteSQL("history");

			getClipboardList();
		});

		listen(LISTEN_KEY.CHANGE_DATA_FILE, getClipboardList);

		// 监听监听状态变更
		listen<boolean>(LISTEN_KEY.TOGGLE_LISTENING, ({ payload }) => {
			if (payload) {
				startListen();
			} else {
				stopListen();
			}
		});

		// 监听全局配置变更
		listen(LISTEN_KEY.GLOBAL_STORE_CHANGED, ({ payload }) => {
			if (isEqual(globalStore, payload)) return;

			merge(globalStore, payload);
		});

		// 监听剪切板配置变更
		listen(LISTEN_KEY.CLIPBOARD_STORE_CHANGED, ({ payload }) => {
			if (isEqual(clipboardStore, payload)) return;

			merge(clipboardStore, payload);
		});

		// 监听主窗口显示/隐藏
		listen(LISTEN_KEY.TOGGLE_MAIN_WINDOW_VISIBLE, toggleWindowVisible);
	});

	// 监听窗口焦点
	useFocus({
		onBlur() {
			if (state.pin) return;

			hideWindow();
		},
	});

	// 监听快捷键
	useRegister(toggleWindowVisible, [shortcut.clipboard]);

	// 获取剪切板内容
	const getClipboardList = async () => {
		const { search, group, favorite } = state;

		const list = await selectSQL<ClipboardItem[]>("history", {
			search,
			group,
			favorite,
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
				if (item.favorite) continue;

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
