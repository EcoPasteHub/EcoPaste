import { WINDOW_PLUGIN } from "@/constants";
import type { WindowLabel } from "@/types/plugin";
import { invoke } from "@tauri-apps/api/core";
import { emit } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import {
	LogicalPosition,
	LogicalSize,
	currentMonitor,
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

	if (focused) {
		hideWindow();
	} else {
		if (appWindow.label === WINDOW_LABEL.MAIN) {
			const { window } = clipboardStore;

			// 激活时回到顶部
			if (window.backTop) {
				await emit(LISTEN_KEY.ACTIVATE_BACK_TOP);
			}

			if (window.style === "float" && window.position !== "remember") {
				const current = await currentMonitor();
				const monitor = await getCursorMonitor();

				if (current && monitor) {
					let { position, size, cursorX, cursorY } = monitor;
					const windowSize = await appWindow.innerSize();
					const { width, height } = windowSize.toLogical(current.scaleFactor);

					if (window.position === "follow") {
						cursorX = Math.min(cursorX, position.x + size.width - width);
						cursorY = Math.min(cursorY, position.y + size.height - height);
					} else {
						cursorX = position.x + (size.width - width) / 2;
						cursorY = position.y + (size.height - height) / 2;
					}

					await appWindow.setPosition(
						new LogicalPosition(Math.round(cursorX), Math.round(cursorY)),
					);
				}
			} else if (window.style === "dock") {
				const monitor = await getCursorMonitor();

				if (monitor) {
					const { width, height } = monitor.size;
					const windowHeight = 400;
					const { x } = monitor.position;
					const y = height - windowHeight;

					await appWindow.setSize(new LogicalSize(width, windowHeight));
					await appWindow.setPosition(new LogicalPosition(x, y));
				}
			}
		}

		showWindow();
	}
};

/**
 * 显示任务栏图标
 */
export const showTaskbarIcon = (show = true) => {
	invoke(WINDOW_PLUGIN.SHOW_TASKBAR_ICON, { show });
};
