import type { OsType } from "@tauri-apps/api/os";

export type Theme = "auto" | "light" | "dark";

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
}

export interface ClipboardStore {
	wakeUpKey: string;
	enableAudio?: boolean;
	historyCapacity: number;
	activeIndex: number;
	windowPosition: "default" | "follow" | "center";
	searchPosition: "top" | "bottom";
	doubleClickFeedback: "none" | "copy";
	replaceAllImagePath?: boolean;
	isFocus: boolean;
	defaultFocus: "firstItem" | "search";
	saveImageDir: string;
}
