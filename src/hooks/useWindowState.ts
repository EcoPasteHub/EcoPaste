import { PhysicalPosition, PhysicalSize } from "@tauri-apps/api/dpi";
import type { Event } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { exists, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";
import { useMount, useReactive } from "ahooks";
import { getSaveWindowStatePath } from "@/utils/path";
import { useTauriFocus } from "./useTauriFocus";

const appWindow = getCurrentWebviewWindow();
const { label } = appWindow;

export const useWindowState = () => {
  const state = useReactive<Partial<PhysicalPosition & PhysicalSize>>({});

  useMount(() => {
    appWindow.onMoved(onChange);

    appWindow.onResized(onChange);
  });

  useTauriFocus({
    onBlur() {
      saveState();
    },
  });

  const onChange = async (event: Event<PhysicalPosition | PhysicalSize>) => {
    const minimized = await appWindow.isMinimized();

    if (minimized) return;

    Object.assign(state, event.payload);
  };

  const getSavedStates = async () => {
    const path = await getSaveWindowStatePath();

    const existed = await exists(path);

    if (!existed) return {};

    const states = await readTextFile(path);

    return JSON.parse(states);
  };

  const saveState = async () => {
    const path = await getSaveWindowStatePath();

    const states = await getSavedStates();

    states[label] = state;

    return writeTextFile(path, JSON.stringify(states, null, 2));
  };

  const restoreState = async () => {
    const states = await getSavedStates();

    Object.assign(state, states[label]);

    const { x, y, width, height } = state;

    if (x && y) {
      appWindow.setPosition(new PhysicalPosition(x, y));
    }

    if (width && height) {
      appWindow.setSize(new PhysicalSize(width, height));
    }
  };

  return {
    restoreState,
    saveState,
  };
};
