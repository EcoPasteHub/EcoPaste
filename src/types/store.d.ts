export type Theme = "auto" | "light" | "dark";

export type Language = (typeof LANGUAGE)[keyof typeof LANGUAGE];

export interface Store {
	globalStore: GlobalStore;
	clipboardStore: ClipboardStore;
}

export interface GlobalStore {
	// 应用设置
	wakeUpKey: string;
	autoStart: boolean;
	autoUpdate: boolean;

	// 外观设置
	theme: Theme;
	language?: Language;

	// 非配置项
	appInfo: { name: string; version: string };
	platform?: OsType;
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
	searchAutoClear: boolean;

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
