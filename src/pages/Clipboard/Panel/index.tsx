import type { AudioRef } from "@/components/Audio";
import Audio from "@/components/Audio";
import { getWebShortcuts } from "@/components/ProShortcut/keys";
import type { HistoryTablePayload, TablePayload } from "@/types/database";
import type { Store } from "@/types/store";
import { listen } from "@tauri-apps/api/event";
import type { EventEmitter } from "ahooks/lib/useEventEmitter";
import { find, findIndex, isNil, last, merge, range } from "lodash-es";
import { nanoid } from "nanoid";
import { createContext } from "react";
import { useSnapshot } from "valtio";
import Dock from "./components/Dock";
import Float from "./components/Float";

interface State extends TablePayload {
	pin?: boolean;
	list: HistoryTablePayload[];
	activeId?: string;
	eventBusId?: string;
	$eventBus?: EventEmitter<string>;
	quickPasteKeys: string[];
}

const INITIAL_STATE: State = {
	list: [],
	quickPasteKeys: [],
};

interface ClipboardPanelContextValue {
	state: State;
	getList?: (payload?: HistoryTablePayload) => Promise<void>;
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

			const createTime = formatDate();

			if (findItem) {
				const { id } = findItem;

				const index = findIndex(state.list, { id });

				const [targetItem] = state.list.splice(index, 1);

				state.list.unshift({ ...targetItem, createTime });

				updateSQL("history", { id, createTime });
			} else {
				const data: HistoryTablePayload = {
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

		// 监听配置项变化
		listen<Store>(LISTEN_KEY.STORE_CHANGED, ({ payload }) => {
			merge(globalStore, payload.globalStore);
			merge(clipboardStore, payload.clipboardStore);
		});

		// 监听快速粘贴的启用状态变更
		subscribeKey(
			globalStore.shortcut.quickPaste,
			"enable",
			setQuickPasteKeys,
			true,
		);

		// 监听快速粘贴的快捷键变更
		subscribeKey(globalStore.shortcut.quickPaste, "value", setQuickPasteKeys);

		// 监听是否显示任务栏图标
		subscribeKey(globalStore.app, "showTaskbarIcon", showTaskbarIcon, true);

		// 切换剪贴板监听状态
		listen<boolean>(LISTEN_KEY.TOGGLE_LISTEN_CLIPBOARD, ({ payload }) => {
			toggleListen(payload);
		});
	});

	// 监听窗口焦点
	useTauriFocus({
		onBlur() {
			if (state.pin) return;

			hideWindow();
		},
	});

	// 监听窗口显隐的快捷键
	useRegister(toggleWindowVisible, [shortcut.clipboard]);

	// 监听粘贴为纯文本的快捷键
	useKeyPress(getWebShortcuts(shortcut.pastePlain), (event) => {
		event.preventDefault();

		const data = find(state.list, { id: state.activeId });

		pasteClipboard(data, true);
	});

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
		const { group, search, favorite } = state;

		state.list = await selectSQL<HistoryTablePayload[]>("history", {
			group,
			search,
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
		<>
			<Audio hiddenIcon ref={audioRef} />

			<ClipboardPanelContext.Provider
				value={{
					state,
					getList,
				}}
			>
				{window.style === "float" ? <Float /> : <Dock />}
			</ClipboardPanelContext.Provider>
		</>
	);
};

export default ClipboardPanel;
