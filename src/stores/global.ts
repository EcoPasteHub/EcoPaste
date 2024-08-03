import type { GlobalStore } from "@/types/store";
import { getName, getVersion } from "@tauri-apps/api/app";
import { type } from "@tauri-apps/api/os";
import proxyWithPersist, {
	PersistStrategy,
	type ProxyPersistStorageEngine,
} from "valtio-persist";
import { subscribeKey } from "valtio/utils";

export const getStorage = (): ProxyPersistStorageEngine => ({
	getItem: (name) => localStorage.getItem(name),
	setItem: (name, value) => localStorage.setItem(name, value),
	removeItem: (name) => localStorage.removeItem(name),
	getAllKeys: () => Object.keys(localStorage),
});

export const persistStrategies = PersistStrategy.MultiFile;

export const GLOBAL_STORE_INITIAL_STATE: GlobalStore = {
	wakeUpKey: "Alt+X",
	autoStart: false,
	autoUpdate: false,

	theme: "auto",

	appInfo: { name: "", version: "" },
};

export const globalStore = proxyWithPersist<GlobalStore>({
	name: "global",
	initialState: { ...GLOBAL_STORE_INITIAL_STATE },
	persistStrategies,
	version: 0,
	migrations: {},
	getStorage,
});

subscribeKey(globalStore._persist, "loaded", async (loaded) => {
	if (!loaded) return;

	globalStore.appInfo = {
		name: await getName(),
		version: await getVersion(),
	};

	globalStore.platform = await type();

	globalStore.language ??= await getLocale();
});
