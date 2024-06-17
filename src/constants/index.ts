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
	EXISTS: "plugin:fs-extra|exists",
	METADATA: "plugin:fs-extra|metadata",
	GET_IMAGE_BASE64: "plugin:fs-extra|get_image_base64",
};

export const CLIPBOARD_PLUGIN = {
	START_LISTEN: "plugin:clipboard-111|start_listen",
	STOP_LISTEN: "plugin:clipboard-111|stop_listen",
	HAS_FILES: "plugin:clipboard-111|has_files",
	HAS_IMAGE: "plugin:clipboard-111|has_image",
	HAS_HTML: "plugin:clipboard-111|has_html",
	HAS_RICH_TEXT: "plugin:clipboard-111|has_rich_text",
	HAS_TEXT: "plugin:clipboard-111|has_text",
	READ_FILES: "plugin:clipboard-111|read_files",
	READ_IMAGE: "plugin:clipboard-111|read_image",
	READ_HTML: "plugin:clipboard-111|read_html",
	READ_RICH_TEXT: "plugin:clipboard-111|read_rich_text",
	READ_TEXT: "plugin:clipboard-111|read_text",
	WRITE_FILES: "plugin:clipboard-111|write_files",
	WRITE_IMAGE: "plugin:clipboard-111|write_image",
	WRITE_HTML: "plugin:clipboard-111|write_html",
	WRITE_RICH_TEXT: "plugin:clipboard-111|write_rich_text",
	WRITE_TEXT: "plugin:clipboard-111|write_text",
	CLIPBOARD_UPDATE: "plugin:clipboard-111://clipboard_update",
};
