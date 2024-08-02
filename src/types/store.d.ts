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

export type ClickFeedback = "none" | "copy" | "paste";

export interface ClipboardStore {
	// 窗口设置
	wakeUpKey: string;
	windowPosition: "default" | "follow" | "center";

	// 音效设置
	copyAudio: boolean;

	// 搜索框设置
	searchPosition: "top" | "bottom";
	searchDefaultFocus: boolean;

	// 点击反馈
	singleClick: ClickFeedback;
	doubleClick: ClickFeedback;

	// 历史记录
	historyDuration: number;

	// 非配置项
	saveImageDir: string;

	// 版本迭代，后续删除
	replaceSettings?: boolean;
	enableAudio?: boolean;
	historyCapacity?: number;
	doubleClickFeedback?: ClickFeedback;
	defaultFocus?: "first" | "search";
}
