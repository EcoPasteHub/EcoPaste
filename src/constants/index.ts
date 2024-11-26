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
	ACTIVATE_BACK_TOP: "activate-back-top",
};

export const WINDOW_PLUGIN = {
	SHOW_WINDOW: "plugin:eco-window|show_window",
	HIDE_WINDOW: "plugin:eco-window|hide_window",
	SHOW_TASKBAR_ICON: "plugin:eco-window|show_taskbar_icon",
};

export const CLIPBOARD_PLUGIN = {
	START_LISTEN: "plugin:eco-clipboard|start_listen",
	STOP_LISTEN: "plugin:eco-clipboard|stop_listen",
	HAS_FILES: "plugin:eco-clipboard|has_files",
	HAS_IMAGE: "plugin:eco-clipboard|has_image",
	HAS_HTML: "plugin:eco-clipboard|has_html",
	HAS_RTF: "plugin:eco-clipboard|has_rtf",
	HAS_TEXT: "plugin:eco-clipboard|has_text",
	READ_FILES: "plugin:eco-clipboard|read_files",
	READ_IMAGE: "plugin:eco-clipboard|read_image",
	READ_HTML: "plugin:eco-clipboard|read_html",
	READ_RTF: "plugin:eco-clipboard|read_rtf",
	READ_TEXT: "plugin:eco-clipboard|read_text",
	WRITE_FILES: "plugin:eco-clipboard|write_files",
	WRITE_IMAGE: "plugin:eco-clipboard|write_image",
	WRITE_HTML: "plugin:eco-clipboard|write_html",
	WRITE_RTF: "plugin:eco-clipboard|write_rtf",
	WRITE_TEXT: "plugin:eco-clipboard|write_text",
	CLIPBOARD_UPDATE: "plugin:eco-clipboard://clipboard_update",
};

export const OCR_PLUGIN = {
	SYSTEM_OCR: "plugin:eco-ocr|system_ocr",
};

export const LOCALE_PLUGIN = {
	GET_LOCALE: "plugin:eco-locale|get_locale",
};

export const PASTE_PLUGIN = {
	PASTE: "plugin:eco-paste|paste",
};

export const AUTOSTART_PLUGIN = {
	IS_AUTOSTART: "plugin:eco-autostart|is_autostart",
};
