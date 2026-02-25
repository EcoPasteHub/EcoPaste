import { proxy } from "valtio";
import type { ClipboardStore } from "@/types/store";

export const clipboardStore = proxy<ClipboardStore>({
  audio: {
    copy: false,
  },

  content: {
    autoFavorite: false,
    autoPaste: "double",
    autoSort: false,
    copyPlain: false,
    defaultCollapse: false,
    deleteConfirm: true,
    displayLines: 4,
    imageDisplayHeight: 100,
    operationButtons: ["copy", "star", "delete"],
    pastePlain: false,
    showOriginalContent: false,
  },

  history: {
    duration: 0,
    maxCount: 0,
    unit: 1,
  },

  search: {
    autoClear: false,
    defaultFocus: false,
    position: "top",
  },
  webdav: {
    autoBackup: 0,
    lastBackupAt: void 0,
    lastBackupError: void 0,
    lastBackupStatus: "none",
    maxBackups: 0,
    slim: false,
  },
  window: {
    backTop: false,
    position: "remember",
    showAll: false,
    style: "standard",
  },
});
