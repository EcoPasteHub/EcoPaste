export const GITHUB_LINK = "https://github.com/ayangweb/EcoCopy";

export const LISTEN_KEY = {
	ABOUT: "about",
	GITHUB: "github",
	GLOBAL_STORE_CHANGED: "global-store-changed",
	CLIPBOARD_STORE_CHANGED: "clipboard-store-changed",
	CLEAR_HISTORY: "clear-history",
};

export const WINDOW_PLUGIN = {
	CREATE_WINDOW: "plugin:window|create_window",
	SHOW_WINDOW: "plugin:window|show_window",
	HIDE_WINDOW: "plugin:window|hide_window",
	QUIT_APP: "plugin:window|quit_app",
};

export const FS_EXTRA_PLUGIN = {
	METADATA: "plugin:fs-extra|metadata",
	GET_IMAGE_BASE64: "plugin:fs-extra|get_image_base64",
	PREVIEW_FILE: "plugin:fs-extra|preview_file",
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
