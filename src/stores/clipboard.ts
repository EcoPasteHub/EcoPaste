import type { ClipboardStore } from "@/types/store";
import { appDataDir, sep } from "@tauri-apps/api/path";
import proxyWithPersist from "valtio-persist";
import { subscribeKey } from "valtio/utils";

export const CLIPBOARD_STORE_INITIAL_STATE: ClipboardStore = {
	wakeUpKey: "Alt+C",
	windowPosition: "default",

	copyAudio: false,

	searchPosition: "top",
	searchDefaultFocus: false,
	searchAutoClear: false,

	singleClick: "none",
	doubleClick: "none",

	historyDuration: 0,

	saveImageDir: "",
};

export const clipboardStore = proxyWithPersist<ClipboardStore>({
	name: "clipboard",
	initialState: { ...CLIPBOARD_STORE_INITIAL_STATE },
	persistStrategies,
	version: 0,
	migrations: {},
	getStorage,
});

subscribeKey(clipboardStore._persist, "loaded", async (loaded) => {
	if (!loaded) return;

	clipboardStore.saveImageDir = `${await appDataDir()}images${sep}`;

	// 版本迭代，后续删除
	if (clipboardStore.replaceSettings) return;

	const { enableAudio, historyCapacity, doubleClickFeedback, defaultFocus } =
		clipboardStore;

	if (enableAudio) {
		clipboardStore.copyAudio = enableAudio;
	}

	if (historyCapacity) {
		clipboardStore.historyDuration = historyCapacity;
	}

	if (doubleClickFeedback) {
		clipboardStore.doubleClick = doubleClickFeedback;
	}

	if (defaultFocus) {
		clipboardStore.searchDefaultFocus = defaultFocus === "search";
	}

	clipboardStore.replaceSettings = true;
});
