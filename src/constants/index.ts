export const GITHUB_LINK = "https://github.com/ayangweb/EcoPaste";

export const GITHUB_ISSUES_LINK = "https://github.com/ayangweb/EcoPaste/issues";

export const LISTEN_KEY = {
	ABOUT: "about",
	GITHUB: "github",
	GLOBAL_STORE_CHANGED: "global-store-changed",
	CLIPBOARD_STORE_CHANGED: "clipboard-store-changed",
	CLEAR_HISTORY: "clear-history",
	UPDATE: "update",
	TRAY_CLICK: "tray-click",
	IMPORT_DATA: "import-data",
	CHANGE_LANGUAGE: "change-language",
};

export const WINDOW_PLUGIN = {
	CREATE_WINDOW: "plugin:window|create_window",
	SHOW_WINDOW: "plugin:window|show_window",
	HIDE_WINDOW: "plugin:window|hide_window",
	SET_WINDOW_SHADOW: "plugin:window|set_window_shadow",
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
	HAS_RICH_TEXT: "plugin:clipboard|has_rich_text",
	HAS_TEXT: "plugin:clipboard|has_text",
	READ_FILES: "plugin:clipboard|read_files",
	READ_IMAGE: "plugin:clipboard|read_image",
	READ_HTML: "plugin:clipboard|read_html",
	READ_RICH_TEXT: "plugin:clipboard|read_rich_text",
	READ_TEXT: "plugin:clipboard|read_text",
	WRITE_FILES: "plugin:clipboard|write_files",
	WRITE_IMAGE: "plugin:clipboard|write_image",
	WRITE_HTML: "plugin:clipboard|write_html",
	WRITE_RICH_TEXT: "plugin:clipboard|write_rich_text",
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
};

export const STORE_FILE_NAME = "store";

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

export const AUTO_LAUNCH_PLUGIN = {
	IS_AUTO_LAUNCH: "plugin:auto_launch|is_auto_launch",
};
