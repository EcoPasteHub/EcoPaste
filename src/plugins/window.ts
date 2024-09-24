import { WINDOW_PLUGIN } from "@/constants";
import type { WindowLabel } from "@/types/plugin";
import { invoke } from "@tauri-apps/api";
import { emit } from "@tauri-apps/api/event";
import {
	PhysicalPosition,
	appWindow,
	availableMonitors,
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
	const focused = await appWindow.isFocused();

	if (appWindow.label === WINDOW_LABEL.MAIN) {
		const { window } = clipboardStore;

		if (window.style === "float") {
			if (!focused && window.position !== "remember") {
				const monitors = await availableMonitors();

				if (!monitors.length) return;

				const { width, height } = await appWindow.innerSize();

				const [x, y] = await getMouseCoords();

				for (const monitor of monitors) {
					const {
						scaleFactor,
						position: { x: posX, y: posY },
						size: { width: screenWidth, height: screenHeight },
					} = monitor;

					const factor = isMac() ? scaleFactor : 1;

					let coordX = x * factor;
					let coordY = y * factor;

					if (
						coordX < posX ||
						coordY < posY ||
						coordX > posX + screenWidth ||
						coordY > posY + screenHeight
					) {
						continue;
					}

					if (window.position === "follow") {
						coordX = Math.min(coordX, posX + screenWidth - width);
						coordY = Math.min(coordY, posY + screenHeight - height);
					} else if (window.position === "center") {
						coordX = posX + (screenWidth - width) / 2;
						coordY = posY + (screenHeight - height) / 2;
					}

					appWindow.setPosition(new PhysicalPosition(coordX, coordY));

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
