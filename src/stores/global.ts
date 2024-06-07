import type { GlobalStore, Theme } from "@/types/store";
import { getName, getVersion } from "@tauri-apps/api/app";
import { appWindow } from "@tauri-apps/api/window";
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

export const globalStore = proxyWithPersist<GlobalStore>({
	name: "global",
	initialState: {
		autoStart: false,
		tabTrigger: "click",
		wakeUpKey: "Alt+X",
	},
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

	globalStore.theme ??= "auto";
});

subscribeKey(globalStore, "theme", async (value = "auto") => {
	let theme: Theme = value;

	if (theme === "auto") {
		theme = (await appWindow.theme()) ?? "light";
	}

	globalStore.isDark = theme === "dark";
});

subscribeKey(globalStore, "isDark", (value) => {
	if (value) {
		document.documentElement.classList.add("dark");
	} else {
		document.documentElement.classList.remove("dark");
	}
});

// subscribe(globalStore, () => {
// 	emit(LISTEN_KEY.GLOBAL_STORE_CHANGED, globalStore);
// });
