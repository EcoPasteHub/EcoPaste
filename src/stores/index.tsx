import type { GlobalStore, Theme } from "@/types/store";
import { getName, getVersion } from "@tauri-apps/api/app";
import { appWindow } from "@tauri-apps/api/window";
import proxyWithPersist, { PersistStrategy } from "valtio-persist";
import { subscribeKey } from "valtio/utils";

export const store = proxyWithPersist<GlobalStore>({
	name: "global",
	initialState: {
		theme: "auto",
		isDark: false,
		autoStart: false,
		tabTrigger: "click",
	},
	persistStrategies: PersistStrategy.MultiFile,
	version: 0,
	migrations: {},
	getStorage: () => ({
		getItem: (name) => localStorage.getItem(name),
		setItem: (name, value) => localStorage.setItem(name, value),
		removeItem: (name) => localStorage.removeItem(name),
		getAllKeys: () => Object.keys(localStorage),
	}),
});

subscribeKey(store._persist, "loaded", async (loaded) => {
	if (!loaded) return;

	const name = await getName();
	const version = await getVersion();

	store.appInfo = {
		name,
		version,
	};
});

subscribeKey(store, "theme", async (value) => {
	let theme: Theme = value;

	if (theme === "auto") {
		theme = (await appWindow.theme()) ?? "light";
	}

	store.isDark = theme === "dark";
});

subscribeKey(store, "isDark", (value) => {
	if (value) {
		document.documentElement.classList.add("dark");
	} else {
		document.documentElement.classList.remove("dark");
	}
});
