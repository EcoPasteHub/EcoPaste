export type Theme = "auto" | "light" | "dark";

export interface GlobalStore {
	theme?: Theme;
	isDark?: boolean;
	autoStart: boolean;
	wakeUpKey: string;
	appInfo?: {
		name: string;
		version: string;
	};
}

export interface ClipboardStore {
	wakeUpKey: string;
	enableAudio?: boolean;
	capacity: number;
}
