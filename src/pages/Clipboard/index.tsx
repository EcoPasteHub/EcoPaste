import { motion } from "motion/react";
import { useSnapshot } from "valtio";
import { useClipboardWindowEditableFocus } from "@/hooks/useClipboardWindowEditableFocus";
import { settingsState } from "@/stores/settings";
import { isClipboardDockLayout } from "@/utils/is";
import Footer from "./components/Footer";
import Group from "./components/Group";
import Header from "./components/Header";
import List from "./components/List";
import { useClipboardDockSlide } from "./hooks/useClipboardDockSlide";

const Clipboard = () => {
  useClipboardWindowEditableFocus();
  const settings = useSnapshot(settingsState);
  const isDock = isClipboardDockLayout(settings.clipboard.window.position);
  const { open, transition } = useClipboardDockSlide(isDock);

  if (!isDock) {
    return (
      <div
        className="flex size-screen flex-col overflow-hidden bg-ant-container"
        data-tauri-drag-region
      >
        <Header />
        <Group />
        <List />
        <Footer />
      </div>
    );
  }

  return (
    <div className="size-screen overflow-hidden">
      <motion.div
        animate={{ y: open ? 0 : "100%" }}
        className="flex size-full flex-col overflow-hidden rounded-t-4 bg-ant-container shadow-lg"
        data-tauri-drag-region
        initial={{ y: "100%" }}
        transition={transition}
      >
        <div
          aria-hidden="true"
          className="flex h-2 shrink-0 items-center justify-center"
        >
          <span className="h-0.5 w-10 rounded-full bg-ant-fill-secondary" />
        </div>
        <div className="flex items-center gap-2 px-3 py-1">
          <Header />
          <div className="min-w-0 flex-1">
            <Group />
          </div>
        </div>
        <List />
      </motion.div>
    </div>
  );
};

export default Clipboard;
