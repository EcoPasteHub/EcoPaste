import { invoke } from "@tauri-apps/api/core";
import { emit } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { PhysicalPosition, PhysicalSize } from "@tauri-apps/api/window";
import { LISTEN_KEY, WINDOW_LABEL } from "@/constants";
import { clipboardStore } from "@/stores/clipboard";
import type { WindowLabel } from "@/types/plugin";

import { getCursorMonitor } from "@/utils/monitor";

const COMMAND = {
  GET_CARET_POSITION: "plugin:eco-window|get_caret_position",
  HIDE_WINDOW: "plugin:eco-window|hide_window",
  SHOW_TASKBAR_ICON: "plugin:eco-window|show_taskbar_icon",
  SHOW_WINDOW: "plugin:eco-window|show_window",
};

interface CaretPosition {
  x: number;
  y: number;
  success: boolean;
}

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

  // 使用 isVisible() 判断窗口状态，因为不夺焦模式下 isFocused() 始终为 false
  const visible = await appWindow.isVisible();

  if (visible) {
    return hideWindow();
  }

  if (appWindow.label === WINDOW_LABEL.MAIN) {
    const { window } = clipboardStore;

    // 激活时回到顶部
    if (window.backTop) {
      await emit(LISTEN_KEY.ACTIVATE_BACK_TOP);
    }

    // 默认收起：激活窗口时清空展开状态
    if (clipboardStore.content.defaultCollapse) {
      await emit(LISTEN_KEY.ACTIVATE_DEFAULT_COLLAPSE);
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
        } else if (window.position === "caret") {
          // 跟随输入光标位置
          try {
            const caretPos = await invoke<CaretPosition>(
              COMMAND.GET_CARET_POSITION,
            );
            if (caretPos.success && caretPos.x > 0 && caretPos.y > 0) {
              x = Math.min(caretPos.x, position.x + size.width - width);
              y = Math.min(caretPos.y + 20, position.y + size.height - height); // +20 偏移避免遮挡光标
            } else {
              // 获取失败时回退到跟随鼠标
              x = Math.min(x, position.x + size.width - width);
              y = Math.min(y, position.y + size.height - height);
            }
          } catch {
            // 获取失败时回退到跟随鼠标
            x = Math.min(x, position.x + size.width - width);
            y = Math.min(y, position.y + size.height - height);
          }
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
