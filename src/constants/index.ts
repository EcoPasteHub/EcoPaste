export const GITHUB_LINK = "https://github.com/EcoPasteHub/EcoPaste";

export const GITHUB_ISSUES_LINK = `${GITHUB_LINK}/issues`;

export const LISTEN_KEY = {
	ABOUT: "about",
	GITHUB: "github",
	GLOBAL_STORE_CHANGED: "global-store-changed",
	CLIPBOARD_STORE_CHANGED: "clipboard-store-changed",
	UPDATE_APP: "update-app",
	TRAY_CLICK: "tray-click",
	REFRESH_CLIPBOARD_LIST: "refresh-clipboard-list",
	CHANGE_LANGUAGE: "change-language",
	TOGGLE_LISTENING: "toggle-listening",
	SHOW_WINDOW: "show-window",
	CLIPBOARD_ITEM_PREVIEW: "clipboard-item-preview",
	CLIPBOARD_ITEM_PASTE: "clipboard-item-paste",
	CLIPBOARD_ITEM_DELETE: "clipboard-item-delete",
	CLIPBOARD_ITEM_SELECT_PREV: "clipboard-item-select-prev",
	CLIPBOARD_ITEM_SELECT_NEXT: "clipboard-item-select-next",
	TOGGLE_MAIN_WINDOW_VISIBLE: "toggle-main-window-visible",
	CLOSE_DATABASE: "close-database",
};

export const WINDOW_PLUGIN = {
	CREATE_WINDOW: "plugin:window|create_window",
	SHOW_WINDOW: "plugin:window|show_window",
	HIDE_WINDOW: "plugin:window|hide_window",
	SHOW_TASKBAR_ICON: "plugin:window|show_taskbar_icon",
};

export const FS_EXTRA_PLUGIN = {
	METADATA: "plugin:fs-extra|metadata",
	PREVIEW_PATH: "plugin:fs-extra|preview_path",
};

export const CLIPBOARD_PLUGIN = {
	START_LISTEN: "plugin:clipboard|start_listen",
	STOP_LISTEN: "plugin:clipboard|stop_listen",
	HAS_FILES: "plugin:clipboard|has_files",
	HAS_IMAGE: "plugin:clipboard|has_image",
	HAS_HTML: "plugin:clipboard|has_html",
	HAS_RTF: "plugin:clipboard|has_rtf",
	HAS_TEXT: "plugin:clipboard|has_text",
	READ_FILES: "plugin:clipboard|read_files",
	READ_IMAGE: "plugin:clipboard|read_image",
	READ_HTML: "plugin:clipboard|read_html",
	READ_RTF: "plugin:clipboard|read_rtf",
	READ_TEXT: "plugin:clipboard|read_text",
	WRITE_FILES: "plugin:clipboard|write_files",
	WRITE_IMAGE: "plugin:clipboard|write_image",
	WRITE_HTML: "plugin:clipboard|write_html",
	WRITE_RTF: "plugin:clipboard|write_rtf",
	WRITE_TEXT: "plugin:clipboard|write_text",
	CLIPBOARD_UPDATE: "plugin:clipboard://clipboard_update",
};

export const MOUSE_PLUGIN = {
	GET_MOUSE_COORDS: "plugin:mouse|get_mouse_coords",
};

export const OCR_PLUGIN = {
	SYSTEM_OCR: "plugin:ocr|system_ocr",
};

export const THEME_PLUGIN = {
	GET_THEME: "plugin:theme|get_theme",
	SET_THEME: "plugin:theme|set_theme",
};

export const BACKUP_PLUGIN = {
	EXPORT_DATA: "plugin:backup|export_data",
	IMPORT_DATA: "plugin:backup|import_data",
	MOVE_DATA: "plugin:backup|move_data",
};

export const LANGUAGE = {
	ZH_CN: "zh-CN",
	ZH_TW: "zh-TW",
	EN_US: "en-US",
	JA_JP: "ja-JP",
} as const;

export const LOCALE_PLUGIN = {
	GET_LOCALE: "plugin:locale|get_locale",
	SET_LOCALE: "plugin:locale|set_locale",
};

export const PASTE_PLUGIN = {
	PASTE: "plugin:paste|paste",
};

export const MACOS_PERMISSIONS_PLUGIN = {
	CHECK_ACCESSIBILITY_PERMISSIONS:
		"plugin:macos-permissions|check_accessibility_permissions",
	REQUEST_ACCESSIBILITY_PERMISSIONS:
		"plugin:macos-permissions|request_accessibility_permissions",
	REQUEST_FULL_DISK_ACCESS_PERMISSIONS:
		"plugin:macos-permissions|request_full_disk_access_permissions",
};

export const TRAY_PLUGIN = {
	SET_TRAY_VISIBLE: "plugin:tray|set_tray_visible",
};

export const WINDOW_LABEL = {
	MAIN: "main",
	PREFERENCE: "preference",
} as const;

export const UPDATER_PLUGIN = {
	CHECK_UPDATE: "plugin:updater|check_update",
};
