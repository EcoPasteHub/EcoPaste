import { sep } from "@tauri-apps/api/path";

/**
 * 获取存储图片的目录
 */
export const getSaveImageDir = () => {
	return `${globalStore.env.saveDataDir}images${sep}`;
};

/**
 * 备份数据时全局数据的存储路径
 */
export const getBackupStorePath = () => {
	return `${globalStore.env.saveDataDir}backup-store`;
};
