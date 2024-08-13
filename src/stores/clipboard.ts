import type { ClipboardStore } from "@/types/store";
import proxyWithPersist from "valtio-persist";

export const CLIPBOARD_STORE_INITIAL_STATE: ClipboardStore = {
	window: {
		position: "default",
	},

	audio: {
		copy: false,
	},

	search: {
		position: "top",
		defaultFocus: false,
		autoClear: false,
	},

	content: {
		autoPaste: "double",
		ocr: true,
		copyPlainText: false,
	},

	history: {
		duration: 0,
		unit: 1,
	},
};

export const clipboardStore = proxyWithPersist<ClipboardStore>({
	name: "clipboard",
	initialState: { ...CLIPBOARD_STORE_INITIAL_STATE },
	persistStrategies,
	version: 0,
	migrations: {},
	getStorage,
});
