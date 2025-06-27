import type { Platform } from "@tauri-apps/plugin-os";

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
		silentStart: boolean;
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
		pastePlain: string;
	};

	// 只在当前系统环境使用
	env: {
		platform?: Platform;
		appName?: string;
		appVersion?: string;
		saveDataDir?: string;
	};
}

export type ClickFeedback = "none" | "copy" | "paste";

export type OperationButton =
	| "copy"
	| "pastePlain"
	| "note"
	| "star"
	| "delete";

export interface ClipboardStore {
	// 窗口设置
	window: {
		style: "float" | "dock";
		position: "remember" | "follow" | "center";
		backTop: boolean;
		showAll: boolean;
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

	// 剪贴板内容设置
	content: {
		autoPaste: "single" | "double";
		ocr: boolean;
		copyPlain: boolean;
		pastePlain: boolean;
		operationButtons: OperationButton[];
		autoFavorite: boolean;
		deleteConfirm: boolean;
		autoSort: boolean;
		showOriginalContent: boolean;
	};

	// 历史记录
	history: {
		duration: number;
		unit: number;
		maxCount: number;
	};

	// 排除的文件类型
	excludeFiles: string[];

	// 文件大小限制
	fileSizeLimit: number;
}
