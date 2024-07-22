import type { ClipboardStore } from "@/types/store";
import { appDataDir, sep } from "@tauri-apps/api/path";
import proxyWithPersist from "valtio-persist";
import { subscribeKey } from "valtio/utils";

export const clipboardStore = proxyWithPersist<ClipboardStore>({
	name: "clipboard",
	initialState: {
		wakeUpKey: "Alt+C",
		historyCapacity: 0,
		windowPosition: "default",
		searchPosition: "top",
		doubleClickFeedback: "none",
		isFocus: false,
		defaultFocus: "firstItem",
		saveImageDir: "",
	},
	persistStrategies,
	version: 0,
	migrations: {},
	getStorage,
});

subscribeKey(clipboardStore._persist, "loaded", async (loaded) => {
	if (!loaded) return;

	clipboardStore.saveImageDir = `${await appDataDir()}images${sep}`;
});
