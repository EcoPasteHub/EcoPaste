import { invoke } from "@tauri-apps/api";
import { camelCase, mapKeys } from "lodash-es";

interface Metadata {
	size: number;
	isDir: boolean;
	isFile: boolean;
	isExist: boolean;
}

const handlePath = (path: string) => {
	return decodeURI(path).replace("file://", "");
};

/**
 * 获取系统文件（夹）的信息
 * @param path 路径
 */
export const metadata = async (path: string) => {
	const result = await invoke<any>("plugin:fs-extra|metadata", {
		path: handlePath(path),
	});

	return mapKeys(result, (_, key) => camelCase(key)) as Metadata;
};

/**
 * 查看文件（夹）是否存在
 * @param path 路径
 */
export const exists = async (path: string) => {
	return await invoke<boolean>("plugin:fs-extra|exists", {
		path: handlePath(path),
	});
};
