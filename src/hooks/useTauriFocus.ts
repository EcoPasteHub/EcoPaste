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

    // Windows 上偶尔会收到“假失焦”事件，这里用 isFocused 二次确认
    const debounced = debounce(async () => {
      const focused = await appWindow.isFocused();

      if (focused) {
        onFocus?.();
      } else {
        onBlur?.();
      }
    }, wait);

    unlistenRef.current = await appWindow.onFocusChanged(() => {
      void debounced();
    });
  });

  useUnmount(() => {
    unlistenRef.current?.();
  });
};
