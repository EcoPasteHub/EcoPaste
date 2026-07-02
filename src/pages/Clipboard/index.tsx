import type { FC } from "react";
import { useSnapshot } from "valtio";
import { settingsState } from "@/stores/settings";
import { cn } from "@/utils/cn";
import Footer from "./components/Footer";
import Group from "./components/Group";
import Header from "./components/Header";
import List from "./components/List";
import { isClipboardBottomSheet, usesClipboardSheetLayout } from "./layout";

const Clipboard: FC = () => {
  const settings = useSnapshot(settingsState);
  const windowPosition = settings.clipboard.window.position;
  const isSheetLayout = usesClipboardSheetLayout(windowPosition);
  const isBottomSheet = isClipboardBottomSheet(windowPosition);

  return (
    <div
      className={cn(
        "flex size-screen flex-col overflow-hidden bg-ant-container",
        {
          "bg-ant-layout": isSheetLayout,
          "rounded-4": isSheetLayout && !isBottomSheet,
          "rounded-t-4": isBottomSheet,
        },
      )}
      data-tauri-drag-region
    >
      <Header />

      <Group />

      <List />

      {isSheetLayout ? null : <Footer />}
    </div>
  );
};

export default Clipboard;
