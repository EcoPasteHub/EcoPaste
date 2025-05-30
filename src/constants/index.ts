export const WEBSITE_LINK = "https://www.ecopaste.cn";

export const GITHUB_LINK = "https://github.com/EcoPasteHub/EcoPaste";

export const GITHUB_ISSUES_LINK = `${GITHUB_LINK}/issues`;

export const TRAY_ID = "app-tray";

export const UPDATE_MESSAGE_KEY = "app-update-message";

export const WINDOW_LABEL = {
	MAIN: "main",
	PREFERENCE: "preference",
} as const;

export const LANGUAGE = {
	ZH_CN: "zh-CN",
	ZH_TW: "zh-TW",
	EN_US: "en-US",
	JA_JP: "ja-JP",
} as const;

export const LISTEN_KEY = {
	STORE_CHANGED: "store-changed",
	UPDATE_APP: "update-app",
	REFRESH_CLIPBOARD_LIST: "refresh-clipboard-list",
	SHOW_WINDOW: "show-window",
	CLIPBOARD_ITEM_PREVIEW: "clipboard-item-preview",
	CLIPBOARD_ITEM_PASTE: "clipboard-item-paste",
	CLIPBOARD_ITEM_DELETE: "clipboard-item-delete",
	CLIPBOARD_ITEM_SELECT_PREV: "clipboard-item-select-prev",
	CLIPBOARD_ITEM_SELECT_NEXT: "clipboard-item-select-next",
	CLOSE_DATABASE: "close-database",
	TOGGLE_LISTEN_CLIPBOARD: "toggle-listen-clipboard",
	CLIPBOARD_ITEM_FAVORITE: "clipboard-item-favorite",
	CLIPBOARD_ITEM_COPY: "clipboard-item-copy",
	ACTIVATE_BACK_TOP: "activate-back-top",
};

export const PRESET_SHORTCUT = {
	SEARCH: isMac ? "meta.f" : "ctrl.f",
	FAVORITE: isMac ? "meta.d" : "ctrl.d",
	OPEN_PREFERENCES: isMac ? "meta.comma" : "ctrl.comma",
	HIDE_WINDOW: isMac ? "meta.w" : "ctrl.w",
	FIXED_WINDOW: isMac ? "meta.p" : "ctrl.p",
	COPY: isMac ? "meta.c" : "ctrl.c",
};
