import type { Metadata } from "@/types/plugin";
import { invoke } from "@tauri-apps/api/core";

/**
 * 获取系统文件（夹）的信息
 * @param path 路径
 */
export const metadata = (path: string) => {
	return invoke<Metadata>(FS_EXTRA_PLUGIN.METADATA, {
		path,
	});
};

/**
 * 预览文件
 * @param path 文件路径
 * @param finder 是否在 finder（文件资源管理器） 中打开，false 是用文件默认的程序打开
 */
export const previewPath = (path: string, finder = true) => {
	return invoke(FS_EXTRA_PLUGIN.PREVIEW_PATH, {
		path,
		finder,
	});
};
