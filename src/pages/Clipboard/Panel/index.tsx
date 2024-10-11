import type { AudioRef } from "@/components/Audio";
import Audio from "@/components/Audio";
import type { ClipboardItem, TablePayload } from "@/types/database";
import { listen } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import type { EventEmitter } from "ahooks/lib/useEventEmitter";
import { find, findIndex, isEqual, isNil, last, merge, range } from "lodash-es";
import { nanoid } from "nanoid";
import { createContext } from "react";
import { useSnapshot } from "valtio";
import { subscribeKey } from "valtio/utils";
import Dock from "./components/Dock";
import Float from "./components/Float";

interface State extends TablePayload {
	pin?: boolean;
	list: ClipboardItem[];
	activeId?: string;
	eventBusId?: string;
	$eventBus?: EventEmitter<string>;
	quickPasteKeys: string[];
	scrollToIndex?: (index: number) => void;
}

const INITIAL_STATE: State = {
	list: [],
	quickPasteKeys: [],
};

interface ClipboardPanelContextValue {
	state: State;
	getList?: (payload?: ClipboardItem) => Promise<void>;
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

	useMount(() => {
		state.$eventBus = $eventBus;

		// 开启剪贴板监听
		startListen();

		// 监听剪切板更新
		onClipboardUpdate((payload) => {
			if (clipboardStore.audio.copy) {
				audioRef.current?.play();
			}

			const { type, value, group } = payload;

			const findItem = find(state.list, { type, value });

			const createTime = dayjs().format("YYYY-MM-DD HH:mm:ss");

			if (findItem) {
				const { id } = findItem;

				const index = findIndex(state.list, { id });

				const [targetItem] = state.list.splice(index, 1);

				state.list.unshift({ ...targetItem, createTime });

				updateSQL("history", { id, createTime });
			} else {
				const data: ClipboardItem = {
					...payload,
					createTime,
					id: nanoid(),
					favorite: false,
				};

				if (state.group === group || (isNil(state.group) && !state.favorite)) {
					state.list.unshift(data);
				}

				insertSQL("history", data);
			}
		});

		// 监听刷新列表
		listen(LISTEN_KEY.REFRESH_CLIPBOARD_LIST, getList);

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

		// 监听快速粘贴的启用状态变更
		watchKey(globalStore.shortcut.quickPaste, "enable", setQuickPasteKeys);

		// 监听快速粘贴的快捷键变更
		subscribeKey(globalStore.shortcut.quickPaste, "value", setQuickPasteKeys);

		// 监听是否显示任务栏图标
		watchKey(globalStore.app, "showTaskbarIcon", showTaskbarIcon);

		// 切换剪贴板监听状态
		listen<boolean>(LISTEN_KEY.TOGGLE_LISTEN_CLIPBOARD, ({ payload }) => {
			toggleListen(payload);
		});
	});

	// 监听窗口焦点
	useFocus({
		onBlur() {
			if (state.pin) return;

			hideWindow();
		},
	});

	// 监听窗口显隐的快捷键
	useRegister(toggleWindowVisible, [shortcut.clipboard]);

	// 监听粘贴为纯文本的快捷键
	useRegister(async () => {
		const focused = await getCurrentWebviewWindow().isFocused();

		if (!focused) return;

		const data = find(state.list, { id: state.activeId });

		pasteClipboard(data, true);
	}, [shortcut.pastePlain]);

	// 监听快速粘贴的快捷键
	useRegister(
		async (event) => {
			if (!globalStore.shortcut.quickPaste.enable) return;

			const index = Number(last(event.shortcut));

			const data = state.list[index - 1];

			pasteClipboard(data);
		},
		[state.quickPasteKeys],
	);

	// 获取剪切板内容
	const getList = async () => {
		const { search, group, favorite } = state;

		state.list = await selectSQL<ClipboardItem[]>("history", {
			search,
			group,
			favorite,
		});
	};

	// 设置快捷粘贴的快捷键
	const setQuickPasteKeys = () => {
		const { enable, value } = globalStore.shortcut.quickPaste;

		if (!enable) {
			state.quickPasteKeys = [];

			return;
		}

		state.quickPasteKeys = range(1, 10).map((item) => [value, item].join("+"));
	};

	return (
		<div>
			{!isLinux() && <Audio hiddenIcon ref={audioRef} />}

			<ClipboardPanelContext.Provider
				value={{
					state,
					getList,
				}}
			>
				{window.style === "float" ? <Float /> : <Dock />}
			</ClipboardPanelContext.Provider>
		</div>
	);
};

export default ClipboardPanel;
