import { invoke } from "@tauri-apps/api/core";
import { homeDir } from "@tauri-apps/api/path";
import { error as logError } from "@tauri-apps/plugin-log";

/**
 * 检查辅助功能权限
 */
export const checkAccessibilityPermissions = () => {
	return invoke<boolean>(
		MACOS_PERMISSIONS_PLUGIN.CHECK_ACCESSIBILITY_PERMISSIONS,
	);
};

/**
 * 请求辅助功能权限
 */
export const requestAccessibilityPermissions = () => {
	return invoke<boolean>(
		MACOS_PERMISSIONS_PLUGIN.REQUEST_ACCESSIBILITY_PERMISSIONS,
	);
};

/**
 * 检查完全磁盘访问权限
 */
export const checkFullDiskAccessPermissions = async () => {
	if (!isMac()) return true;

	try {
		// https://github.com/inket/FullDiskAccess/blob/846e04ea2b84fce843f47d7e7f3421189221829c/Sources/FullDiskAccess/FullDiskAccess.swift#L46
		const checkDirs = ["Library/Containers/com.apple.stocks", "Library/Safari"];

		for await (const dir of checkDirs) {
			const { size } = await metadata(joinPath(await homeDir(), dir));

			if (size === 0) continue;

			return true;
		}

		return false;
	} catch (error) {
		logError(JSON.stringify(error));

		return false;
	}
};

/**
 * 请求完全磁盘访问权限
 */
export const requestFullDiskAccessPermissions = () => {
	return invoke<boolean>(
		MACOS_PERMISSIONS_PLUGIN.REQUEST_FULL_DISK_ACCESS_PERMISSIONS,
	);
};
