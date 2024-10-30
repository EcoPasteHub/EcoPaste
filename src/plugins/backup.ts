import { invoke } from "@tauri-apps/api/core";
import { emit } from "@tauri-apps/api/event";

/**
 * 导出数据
 */
export const exportData = async (dstPath: string) => {
	return invoke(BACKUP_PLUGIN.EXPORT_DATA, {
		dstPath,
		srcDir: getSaveDataDir(),
	});
};

/**
 * 导入数据
 */
export const importData = async (srcPath: string) => {
	await emit(LISTEN_KEY.CLOSE_DATABASE);

	return invoke(BACKUP_PLUGIN.IMPORT_DATA, {
		srcPath,
		dstDir: getSaveDataDir(),
	});
};

/**
 * 移动数据
 * @param from 源文件夹
 * @param to 目标文件夹
 */
export const moveData = async (from: string, to: string) => {
	await emit(LISTEN_KEY.CLOSE_DATABASE);

	return invoke<string>(BACKUP_PLUGIN.MOVE_DATA, { from, to });
};
