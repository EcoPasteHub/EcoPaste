import { invoke } from "@tauri-apps/api";
import { getName } from "@tauri-apps/api/app";
import { open } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { writeFile } from "@tauri-apps/api/fs";
import { omit } from "lodash-es";

/**
 * 备份数据的扩展名
 */
const getExtension = async () => {
	const appName = await getName();

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

	await writeFile(getBackupStorePath(), JSON.stringify(content));

	const time = dayjs().format("YYYY_MM_DD_HH_mm_ss");
	const extension = await getExtension();

	return invoke(BACKUP_PLUGIN.EXPORT_DATA, {
		srcDir: getSaveDataDir(),
		fileName: `${time}.${extension}`,
	});
};

/**
 * 导入数据
 */
export const importData = async () => {
	const extension = await getExtension();

	const path = await open({
		filters: [{ name: "", extensions: [extension] }],
	});

	if (!path) return;

	await emit(LISTEN_KEY.CLOSE_DATABASE);

	return invoke(BACKUP_PLUGIN.IMPORT_DATA, {
		dstDir: getSaveDataDir(),
		path,
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
