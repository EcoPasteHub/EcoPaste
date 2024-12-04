import type { Store } from "@/types/store";
import { getName, getVersion } from "@tauri-apps/api/app";
import { appDataDir } from "@tauri-apps/api/path";
import { exists, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";
import { type } from "@tauri-apps/plugin-os";
import { omit } from "lodash-es";

/**
 * 初始化配置项
 */
const initStore = async () => {
	globalStore.appearance.language ??= await getLocale();
	globalStore.env.platform = await type();
	globalStore.env.appName = await getName();
	globalStore.env.appVersion = await getVersion();
	globalStore.env.saveDataDir ??= await appDataDir();
};

/**
 * 本地存储配置项
 * @param backup 是否为备份数据
 */
export const saveStore = async (backup = false) => {
	const store = { globalStore, clipboardStore };

	const path = await getSaveStorePath(backup);

	return writeTextFile(path, JSON.stringify(store, null, 2));
};

/**
 * 从本地存储恢复配置项
 * @param backup 是否为备份数据
 */
export const restoreStore = async (backup = false) => {
	const path = await getSaveStorePath(backup);

	const existed = await exists(path);

	if (existed) {
		const content = await readTextFile(path);
		const store: Store = JSON.parse(content);
		const nextGlobalStore = omit(store.globalStore, backup ? "env" : "");

		merge(globalStore, nextGlobalStore);
		merge(clipboardStore, store.clipboardStore);
	}

	if (backup) return;

	return initStore();
};
