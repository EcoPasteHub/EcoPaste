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
		showMenubarIcon: boolean;
		showTaskbarIcon: boolean;
	};

	// 外观设置
	appearance: {
		theme: Theme;
		isDark: boolean;
		language?: Language;
	};

	update: {
		auto: boolean;
		beta: boolean;
	};

	// 快捷键设置
	shortcut: {
		clipboard: string;
		preference?: string;
		quickPaste: {
			enable: boolean;
			value: string;
		};
		pastePlain?: string;
	};

	// 只在当前系统环境使用
	env: {
		platform?: OsType;
		appName?: string;
		appVersion?: string;
		saveDataDir?: string;
	};
}

export type ClickFeedback = "none" | "copy" | "paste";

export interface ClipboardStore {
	// 窗口设置
	window: {
		style: "float" | "dock";
		position: "remember" | "follow" | "center";
		backTop: boolean;
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
		copyPlain: boolean;
		pastePlain: boolean;
	};

	// 历史记录
	history: {
		duration: number;
		unit: number;
	};
}
