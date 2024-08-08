export type Theme = "auto" | "light" | "dark";

export type Language = (typeof LANGUAGE)[keyof typeof LANGUAGE];

export interface Store {
	globalStore: GlobalStore;
	clipboardStore: ClipboardStore;
}

export interface GlobalStore {
	// 应用设置
	app: {
		autoStart: boolean;
		autoUpdate: boolean;
	};

	// 外观设置
	appearance: {
		theme: Theme;
		language?: Language;
	};

	// 快捷键设置
	shortcut: {
		clipboard: string;
		preference?: string;
		paste?: string;
	};

	// 非配置项，只提供给全局使用
	env: {
		platform?: OsType;
		appName?: string;
		appVersion?: string;
		saveImageDir?: string;
	};
}

export type ClickFeedback = "none" | "copy" | "paste";

export interface ClipboardStore {
	// 窗口设置
	window: {
		position: "default" | "follow" | "center";
	};

	// 音效设置
	audio: {
		copy: boolean;
	};

	// 搜索框设置
	search: {
		position: "top" | "bottom";
		defaultFocus: boolean;
		autoClear: boolean;
	};

	// 剪切板内容设置
	content: {
		autoPaste: "single" | "double";
		ocr: boolean;
	};

	// 历史记录
	history: {
		duration: number;
	};
}
