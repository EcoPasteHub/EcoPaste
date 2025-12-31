import { invoke } from "@tauri-apps/api/core";
import { emit } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { PhysicalPosition, PhysicalSize } from "@tauri-apps/api/window";
import { LISTEN_KEY, WINDOW_LABEL } from "@/constants";
import { clipboardStore } from "@/stores/clipboard";
import type { WindowLabel } from "@/types/plugin";
import { isLinux } from "@/utils/is";
import { getCursorMonitor } from "@/utils/monitor";

const COMMAND = {
  HIDE_WINDOW: "plugin:eco-window|hide_window",
  SHOW_TASKBAR_ICON: "plugin:eco-window|show_taskbar_icon",
  SHOW_WINDOW: "plugin:eco-window|show_window",
};

/**
 * 显示窗口
 */
export const showWindow = (label?: WindowLabel) => {
  if (label) {
    emit(LISTEN_KEY.SHOW_WINDOW, label);
  } else {
    invoke(COMMAND.SHOW_WINDOW);
  }
};

/**
 * 隐藏窗口
 */
export const hideWindow = () => {
  invoke(COMMAND.HIDE_WINDOW);
};

/**
 * 切换窗口的显示和隐藏
 */
export const toggleWindowVisible = async () => {
  const appWindow = getCurrentWebviewWindow();

  let focused = await appWindow.isFocused();

  if (isLinux) {
    focused = await appWindow.isVisible();
  }

  if (focused) {
    return hideWindow();
  }

  if (appWindow.label === WINDOW_LABEL.MAIN) {
    const { window } = clipboardStore;

    // 激活时回到顶部
    if (window.backTop) {
      await emit(LISTEN_KEY.ACTIVATE_BACK_TOP);
    }

    // 激活时切换至全部分组
    if (window.showAll) {
      await emit(LISTEN_KEY.ACTIVATE_SHOW_ALL);
    }

    if (window.style === "standard" && window.position !== "remember") {
      const monitor = await getCursorMonitor();

      if (monitor) {
        const { position, size, cursorPoint } = monitor;
        const { width, height } = await appWindow.innerSize();
        let { x, y } = cursorPoint;

        if (window.position === "follow") {
          x = Math.min(x, position.x + size.width - width);
          y = Math.min(y, position.y + size.height - height);
        } else {
          x = position.x + (size.width - width) / 2;
          y = position.y + (size.height - height) / 2;
        }

        await appWindow.setPosition(
          new PhysicalPosition(Math.round(x), Math.round(y)),
        );
      }
    } else if (window.style === "dock") {
      const monitor = await getCursorMonitor();

      if (monitor) {
        const { width, height } = monitor.size;
        const { x } = monitor.position;
        const windowHeight = 400;
        const y = height - windowHeight;

        await appWindow.setSize(new PhysicalSize(width, windowHeight));
        await appWindow.setPosition(new PhysicalPosition(x, y));
      }
    }
  }

  showWindow();
};

/**
 * 显示任务栏图标
 */
export const showTaskbarIcon = (visible = true) => {
  invoke(COMMAND.SHOW_TASKBAR_ICON, { visible });
};
