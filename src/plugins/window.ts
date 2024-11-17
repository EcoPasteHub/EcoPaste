import { WINDOW_PLUGIN } from "@/constants";
import type { WindowLabel } from "@/types/plugin";
import { invoke } from "@tauri-apps/api/core";
import { emit } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import {
	PhysicalPosition,
	availableMonitors,
	cursorPosition,
} from "@tauri-apps/api/window";

/**
 * 显示窗口
 */
export const showWindow = (label?: WindowLabel) => {
	if (label) {
		emit(LISTEN_KEY.SHOW_WINDOW, label);
	} else {
		invoke(WINDOW_PLUGIN.SHOW_WINDOW);
	}
};

/**
 * 隐藏窗口
 */
export const hideWindow = () => {
	invoke(WINDOW_PLUGIN.HIDE_WINDOW);
};

/**
 * 切换窗口的显示和隐藏
 */
export const toggleWindowVisible = async () => {
	const appWindow = getCurrentWebviewWindow();

	let focused = await appWindow.isFocused();

	if (isLinux()) {
		focused = await appWindow.isVisible();
	}

	if (appWindow.label === WINDOW_LABEL.MAIN) {
		const { window } = clipboardStore;

		// 激活时回到顶部
		if (window.backTop) {
			await emit(LISTEN_KEY.ACTIVATE_BACK_TOP);
		}

		if (window.style === "float") {
			if (!focused && window.position !== "remember") {
				const monitors = await availableMonitors();

				if (!monitors.length) return;

				const { width, height } = await appWindow.innerSize();

				const cursor = await cursorPosition();

				for await (const monitor of monitors) {
					const { position, size } = monitor;

					let cursorX = cursor.x;
					let cursorY = cursor.y;

					if (
						cursorX < position.x ||
						cursorY < position.y ||
						cursorX > position.x + size.width ||
						cursorY > position.y + size.height
					) {
						continue;
					}

					if (window.position === "follow") {
						cursorX = Math.min(cursorX, position.x + size.width - width);
						cursorY = Math.min(cursorY, position.y + size.height - height);
					} else {
						cursorX = position.x + (size.width - width) / 2;
						cursorY = position.y + (size.height - height) / 2;
					}

					await appWindow.setPosition(
						new PhysicalPosition(Math.round(cursorX), Math.round(cursorY)),
					);

					break;
				}
			}
		} else {
			// TODO: dock 风格的位置
		}
	}

	if (focused) {
		hideWindow();
	} else {
		showWindow();
	}
};

/**
 * 显示任务栏图标
 */
export const showTaskbarIcon = (show = true) => {
	invoke(WINDOW_PLUGIN.SHOW_TASKBAR_ICON, { show });
};
