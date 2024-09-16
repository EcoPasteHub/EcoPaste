import { sep } from "@tauri-apps/api/path";
import { last } from "lodash-es";

/**
 * 连接相应系统的路径
 * @param paths 路径数组
 */
export const joinPath = (...paths: string[]) => {
	const joinPaths = paths.map((path) => {
		if (path.endsWith(sep)) {
			return path.slice(0, -1);
		}

		return path;
	});

	return joinPaths.join(sep);
};

/**
 * 获取存储数据的目录
 * @param endWithSep 结尾是否包含分隔符
 */
export const getSaveDataDir = () => {
	let { saveDataDir = "" } = globalStore.env;

	saveDataDir = joinPath(saveDataDir, "");

	return saveDataDir.slice(0, -1);
};

/**
 * 获取存储图片的目录
 */
export const getSaveImageDir = () => {
	return joinPath(getSaveDataDir(), "images", "");
};

/**
 * 获取图片内容的存储路径
 * @param file 文件名
 */
export const getSaveImagePath = (file: string) => {
	if (file.startsWith(getSaveImageDir())) return file;

	return joinPath(getSaveImageDir(), file);
};

/**
 * 备份数据时全局数据的存储路径
 */
export const getBackupStorePath = () => {
	return joinPath(getSaveDataDir(), "backup-store");
};

/**
 * 存储数据的目录名
 */
export const getSaveDataDirName = () => {
	return last(getSaveDataDir().split(sep)) as string;
};
