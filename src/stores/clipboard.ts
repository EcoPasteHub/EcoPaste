import type { ClipboardStore } from "@/types/store";
import proxyWithPersist from "valtio-persist";

export const clipboardStore = proxyWithPersist<ClipboardStore>({
	name: "clipboard",
	initialState: {
		wakeUpKey: "Alt+C",
		historyCapacity: 0,
		activeIndex: 0,
		windowPosition: "default",
		searchPosition: "top",
		doubleClickFeedback: "none",
	},
	persistStrategies,
	version: 0,
	migrations: {},
	getStorage,
});
