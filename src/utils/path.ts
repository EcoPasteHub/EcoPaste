import { getName } from "@tauri-apps/api/app";
import { appDataDir, sep } from "@tauri-apps/api/path";
import { last } from "lodash-es";

/**
 * 连接相应系统的路径
 * @param paths 路径数组
 */
export const joinPath = (...paths: string[]) => {
	const joinPaths = paths.map((path) => {
		if (path.endsWith(sep())) {
			return path.slice(0, -1);
		}

		return path;
	});

	return joinPaths.join(sep());
};

/**
 * 获取存储数据的目录
 */
export const getSaveDataPath = () => {
	return joinPath(globalStore.env.saveDataDir!);
};

/**
 * 获取数据库文件存储路径
 */
export const getSaveDatabasePath = async () => {
	const appName = await getName();
	const extname = isDev() ? "dev.db" : "db";

	return joinPath(getSaveDataPath(), `${appName}.${extname}`);
};

/**
 * 获取存储图片的路径
 */
export const getSaveImagePath = () => {
	return joinPath(getSaveDataPath(), "images");
};

/**
 * 解析完整的存储图片路径
 * @param file 文件名
 */
export const resolveImagePath = (file: string) => {
	if (file.startsWith(getSaveImagePath())) return file;

	return joinPath(getSaveImagePath(), file);
};

/**
 * 存储数据的目录名
 */
export const getSaveDataDirName = () => {
	return last(getSaveDataPath().split(sep())) as string;
};

/**
 * 存储配置项的路径
 * @param backup 是否是备份数据
 */
export const getSaveStorePath = async (backup = false) => {
	const extname = isDev() ? "dev.json" : "json";

	if (backup) {
		return joinPath(getSaveDataPath(), `.store-backup.${extname}`);
	}

	return joinPath(await appDataDir(), `.store.${extname}`);
};

/**
 * 存储窗口位置的路径
 */
export const getSaveWindowStatePath = async () => {
	const extname = isDev() ? "dev.json" : "json";

	return joinPath(await appDataDir(), `.window-state.${extname}`);
};

/**
 * 获取存储路径系统图标的路径
 */
export const getSaveIconPath = () => {
	return joinPath(getSaveDataPath(), "icons");
};
