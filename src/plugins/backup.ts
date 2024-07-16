import { invoke } from "@tauri-apps/api";
import { getName } from "@tauri-apps/api/app";
import { open } from "@tauri-apps/api/dialog";
import { BaseDirectory, writeFile } from "@tauri-apps/api/fs";
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
		clipboardStore: omit(clipboardStore, ["_persist", "saveImageDir"]),
		globalStore: omit(globalStore, ["_persist", "platform", "appInfo"]),
	};

	await writeFile(STORE_FILE_NAME, JSON.stringify(content), {
		dir: BaseDirectory.AppData,
	});

	const time = dayjs().format("YYYY_MM_DD_HH_mm_ss");
	const extension = await getExtension();

	return invoke(BACKUP_PLUGIN.EXPORT_DATA, {
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

	await closeDatabase();

	return invoke(BACKUP_PLUGIN.IMPORT_DATA, { path });
};
