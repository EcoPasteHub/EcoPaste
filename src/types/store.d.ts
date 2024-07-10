import type { OsType } from "@tauri-apps/api/os";

export type Theme = "auto" | "light" | "dark";

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
	doubleClickFeedback: "none" | "copy";
}
