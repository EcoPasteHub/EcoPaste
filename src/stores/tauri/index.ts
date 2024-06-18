import { getName } from "@tauri-apps/api/app";
import { appDataDir } from "@tauri-apps/api/path";
import { isNil } from "lodash-es";
import { Store } from "tauri-plugin-store-api";

let store: Store;

/**
 * 初始化
 */
export const initStore = async () => {
	const dataDir = await appDataDir();
	const appName = await getName();
	const fileName = isDev() ? "dev.dat" : "dat";

	store = new Store([dataDir, appName, fileName].join("."));
};

/**
 * 获取参数
 * @param key 获取值的属性名
 * @returns 属性值
 */
export const getStore = async <T>(key: string, defaultValue?: T) => {
	const value = await store.get<string>(key);

	return isNil(value) ? defaultValue : JSON.parse(value);
};

/**
 * 写入参数
 * @param key 需要设置的参数的属性名
 * @param value 需要设置的参数的属性值
 */
export const setStore = async (key: string, value: unknown) => {
	if (isNil(value)) {
		return deleteStore(key);
	}

	await store.set(key, JSON.stringify(value));

	store.save();
};

/**
 * 删除参数
 * @param key 要删除的参数的属性名
 */
export const deleteStore = async (key: string) => {
	await store.delete(key);

	await store.save();
};

/**
 * 清空
 */
export const clearStore = async () => {
	await store.clear();

	await store.save();
};
