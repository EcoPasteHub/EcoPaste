import type { OsType } from "@tauri-apps/api/os";

export type Theme = "auto" | "light" | "dark";

export type Language = (typeof LANGUAGE)[keyof typeof LANGUAGE];

export interface Store {
	globalStore: GlobalStore;
	clipboardStore: ClipboardStore;
}

export interface GlobalStore {
	theme: Theme;
	autoStart: boolean;
	wakeUpKey: string;
	appInfo?: { name: string; version: string };
	platform?: OsType;
	autoUpdate?: boolean;
	trayClick: "none" | "show";
	language?: Language;
}

export interface ClipboardStore {
	wakeUpKey: string;
	enableAudio?: boolean;
	historyCapacity: number;
	windowPosition: "default" | "follow" | "center";
	searchPosition: "top" | "bottom";
	clickPaste?: boolean;
	doubleClickFeedback: "none" | "copy" | "paste";
	replaceAllImagePath?: boolean;
	isFocus: boolean;
	defaultFocus: "firstItem" | "search";
	saveImageDir: string;
}
