import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useMount, useUnmount } from "ahooks";
import { debounce } from "es-toolkit";
import { useRef } from "react";
import { isMac } from "@/utils/is";

interface Props {
  onFocus?: () => void;
  onBlur?: () => void;
}

export const useTauriFocus = (props: Props) => {
  const { onFocus, onBlur } = props;
  const unlistenRef = useRef(() => {});

  useMount(async () => {
    const appWindow = getCurrentWebviewWindow();

    const wait = isMac ? 0 : 100;

    const debounced = debounce(({ payload }) => {
      if (payload) {
        onFocus?.();
      } else {
        onBlur?.();
      }
    }, wait);

    unlistenRef.current = await appWindow.onFocusChanged(debounced);
  });

  useUnmount(unlistenRef.current);
};
