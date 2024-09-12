import { sep } from "@tauri-apps/api/path";

/**
 * 获取存储数据的目录
 */
export const getSaveDataDir = () => {
	const { saveDataDir } = globalStore.env;

	if (saveDataDir?.endsWith(sep)) {
		return saveDataDir;
	}

	return saveDataDir + sep;
};

/**
 * 获取存储图片的目录
 */
export const getSaveImageDir = () => {
	return `${getSaveDataDir()}images${sep}`;
};

/**
 * 备份数据时全局数据的存储路径
 */
export const getBackupStorePath = () => {
	return `${getSaveDataDir()}backup-store`;
};
