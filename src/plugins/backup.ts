import { invoke } from "@tauri-apps/api/core";
import { emit } from "@tauri-apps/api/event";
import { downloadDir } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { writeTextFile } from "@tauri-apps/plugin-fs";
import { omit } from "lodash-es";

/**
 * 备份数据的扩展名
 */
const extname = () => {
	const { appName } = globalStore.env;

	return `${appName}-backup`;
};

/**
 * 导出数据
 */
export const exportData = async () => {
	const content = {
		clipboardStore: omit(clipboardStore, "_persist"),
		globalStore: omit(globalStore, ["_persist", "env"]),
	};

	await writeTextFile(getBackupStorePath(), JSON.stringify(content, null, 2));

	const dstPath = joinPath(await downloadDir(), `${formatDate()}.${extname()}`);

	return invoke(BACKUP_PLUGIN.EXPORT_DATA, {
		dstPath,
		srcDir: getSaveDataDir(),
	});
};

/**
 * 导入数据
 */
export const importData = async () => {
	const srcPath = await open({
		filters: [{ name: "", extensions: [extname()] }],
	});

	if (!srcPath) return;

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
