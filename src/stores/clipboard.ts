import type { ClipboardStore } from "@/types/store";
import proxyWithPersist from "valtio-persist";

export const clipboardStore = proxyWithPersist<ClipboardStore>({
	name: "clipboard",
	initialState: {
		wakeUpKey: "Alt+C",
	},
	persistStrategies,
	version: 0,
	migrations: {},
	getStorage,
});

// subscribe(clipboardStore, () => {
// 	emit(LISTEN_KEY.CLIPBOARD_STORE_CHANGED, clipboardStore);
// });
