import { invoke } from "@tauri-apps/api";
import { homeDir } from "@tauri-apps/api/path";
import { error as logError } from "tauri-plugin-log-api";

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

	const homePath = await homeDir();

	try {
		// https://github.com/inket/FullDiskAccess/blob/846e04ea2b84fce843f47d7e7f3421189221829c/Sources/FullDiskAccess/FullDiskAccess.swift#L46
		const checkDirs = ["Library/Containers/com.apple.stocks", "Library/Safari"];

		for (const dir of checkDirs) {
			const { size } = await metadata(`${homePath}${dir}`);

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
