import type { GlobalStore } from "@/types/store";
import { getName, getVersion } from "@tauri-apps/api/app";
import { type } from "@tauri-apps/api/os";
import { appDataDir } from "@tauri-apps/api/path";
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
	app: {
		autoStart: false,
		showMenubarIcon: true,
		showTaskbarIcon: false,
	},

	appearance: {
		theme: "auto",
		isDark: false,
	},

	update: {
		auto: false,
		beta: false,
	},

	shortcut: {
		clipboard: "Alt+C",
		preference: "Alt+X",
		quickPaste: {
			enable: false,
			value: "Command+Shift",
		},
	},

	env: {},
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

	globalStore.appearance.language ??= await getLocale();
	globalStore.env.platform = await type();
	globalStore.env.appName = await getName();
	globalStore.env.appVersion = await getVersion();
	globalStore.env.saveDataDir ??= await appDataDir();
});

// subscribeKey 但是首次使用执行
export const watchKey: typeof subscribeKey = (object, key, callback) => {
	callback(object[key]);

	return subscribeKey(object, key, callback);
};
