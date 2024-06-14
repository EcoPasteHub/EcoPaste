import { invoke } from "@tauri-apps/api";
import { camelCase, mapKeys } from "lodash-es";

interface Metadata {
	size: number;
	isDir: boolean;
	isFile: boolean;
	isExist: boolean;
}

export const metadata = async (path: string) => {
	const result = await invoke<any>("plugin:fs-extra|metadata", {
		path: decodeURI(path),
	});

	return mapKeys(result, (_, key) => camelCase(key)) as Metadata;
};
