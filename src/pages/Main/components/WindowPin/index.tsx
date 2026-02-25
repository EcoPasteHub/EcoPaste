import { listen } from "@tauri-apps/api/event";
import { useKeyPress } from "ahooks";
import clsx from "clsx";
import { useContext, useEffect } from "react";
import UnoIcon from "@/components/UnoIcon";
import { PRESET_SHORTCUT } from "@/constants";
import { useTauriFocus } from "@/hooks/useTauriFocus";
import { hideWindow } from "@/plugins/window";
import { MainContext } from "../..";

const WindowPin = () => {
  const { rootState } = useContext(MainContext);

  useKeyPress(PRESET_SHORTCUT.FIXED_WINDOW, () => {
    togglePin();
  });

  useTauriFocus({
    onBlur() {
      if (rootState.pinned) return;

      hideWindow();
    },
  });

  // 监听窗口外点击事件（不夺焦模式下的自动隐藏）
  useEffect(() => {
    let cleanup: (() => void) | undefined;

    listen("clipboard-outside-click", () => {
      if (rootState.pinned) return;
      hideWindow();
    }).then((unlisten) => {
      cleanup = unlisten;
    });

    return () => {
      cleanup?.();
    };
  }, []);

  const togglePin = () => {
    rootState.pinned = !rootState.pinned;
  };

  return (
    <UnoIcon
      active={rootState.pinned}
      className={clsx({ "-rotate-45": !rootState.pinned })}
      hoverable
      name="i-lets-icons:pin"
      onMouseDown={togglePin}
    />
  );
};

export default WindowPin;
