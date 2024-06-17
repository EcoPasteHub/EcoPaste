import type { Metadata } from "@/types/plugin";
import { invoke } from "@tauri-apps/api";
import { camelCase, mapKeys } from "lodash-es";

/**
 * 查看文件（夹）是否存在
 * @param path 路径
 */
export const exists = async (path: string) => {
	return await invoke<boolean>(FS_EXTRA_PLUGIN.EXISTS, {
		path,
	});
};

/**
 * 获取系统文件（夹）的信息
 * @param path 路径
 */
export const metadata = async (path: string) => {
	const result = await invoke<any>(FS_EXTRA_PLUGIN.METADATA, {
		path,
	});

	return mapKeys(result, (_, key) => camelCase(key)) as Metadata;
};

export const getImageBase64 = async (path: string) => {
	return await invoke<string>(FS_EXTRA_PLUGIN.GET_IMAGE_BASE64, {
		path,
	});
};
