import { cn } from "@heroui/styles";
import { useRef, useState } from "react";
import { useSnapshot } from "valtio";

import { TAURI_EVENT } from "@/constants/events";
import { WINDOW_LABEL } from "@/constants/windows";
import { useTauriListen } from "@/hooks/useTauriListen";
import { clipboardViewState } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";

import ClipboardList from "./components/ClipboardList";
import ClipboardTabs from "./components/ClipboardTabs";
import SearchBar from "./components/SearchBar";

interface VisibilityPayload {
  label: string;
  visible: boolean;
}

const Clipboard = () => {
  const settings = useSnapshot(settingsState);
  const view = useSnapshot(clipboardViewState);
  const search = settings.value?.clipboard.search;
  const position = search?.position ?? "top";
  const [keyword, setKeyword] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  // Rust 端 show_window/hide_window 在 mod.rs 入口处统一 emit，
  // 比依赖平台各异的 tauri://focus/blur 更可靠（Windows 主窗口为不抢焦点设计）。
  useTauriListen<VisibilityPayload>(
    TAURI_EVENT.WINDOW_VISIBILITY,
    (payload) => {
      if (payload.label !== WINDOW_LABEL.MAIN) return;
      if (payload.visible) {
        if (search?.defaultFocus) inputRef.current?.focus();
      } else {
        if (search?.clearOnHide) setKeyword("");
      }
    },
  );

  return (
    <div
      className={cn("flex h-screen w-screen flex-col", {
        "flex-col-reverse": position === "bottom",
      })}
    >
      <SearchBar inputRef={inputRef} onChange={setKeyword} value={keyword} />
      <ClipboardTabs />
      <div className="min-h-0 flex-1">
        <ClipboardList keyword={keyword} tab={view.tab} />
      </div>
    </div>
  );
};

export default Clipboard;
