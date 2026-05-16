import { isMac } from "@/utils/is";

export const WEBSITE_LINK = "https://www.ecopaste.cn";

export const GITHUB_LINK = "https://github.com/EcoPasteHub/EcoPaste";

export const GITHUB_ISSUES_LINK = `${GITHUB_LINK}/issues`;

export const UPDATE_MESSAGE_KEY = "app-update-message";

export const WINDOW_LABEL = {
  MAIN: "main",
  PREFERENCE: "preference",
} as const;

export const LANGUAGE = {
  EN_US: "en-US",
  JA_JP: "ja-JP",
  ZH_CN: "zh-CN",
  ZH_TW: "zh-TW",
} as const;

export const LISTEN_KEY = {
  ACTIVATE_BACK_TOP: "activate-back-top",
  ACTIVATE_SHOW_ALL: "activate-show-all",
  CLIPBOARD_ITEM_DELETE: "clipboard-item-delete",
  CLIPBOARD_ITEM_FAVORITE: "clipboard-item-favorite",
  CLIPBOARD_ITEM_PASTE: "clipboard-item-paste",
  CLIPBOARD_ITEM_PREVIEW: "clipboard-item-preview",
  CLIPBOARD_ITEM_SELECT_NEXT: "clipboard-item-select-next",
  CLIPBOARD_ITEM_SELECT_PREV: "clipboard-item-select-prev",
  CLOSE_DATABASE: "close-database",
  REFRESH_CLIPBOARD_LIST: "refresh-clipboard-list",
  SHOW_WINDOW: "show-window",
  STORE_CHANGED: "store-changed",
  TOGGLE_LISTEN_CLIPBOARD: "toggle-listen-clipboard",
  UPDATE_APP: "update-app",
};

export const PRESET_SHORTCUT = {
  FAVORITE: isMac ? "meta.d" : "ctrl.d",
  FIXED_WINDOW: isMac ? "meta.p" : "ctrl.p",
  HIDE_WINDOW: isMac ? "meta.w" : "ctrl.w",
  OPEN_PREFERENCES: isMac ? "meta.comma" : "ctrl.comma",
  SEARCH: isMac ? "meta.f" : "ctrl.f",
};
