export type Theme = "auto" | "light" | "dark";

export type TabTrigger = "click" | "hover";

export interface GlobalStore {
	theme?: Theme;
	isDark?: boolean;
	autoStart: boolean;
	tabTrigger: TabTrigger;
	wakeUpKey: string;
	appInfo?: {
		name: string;
		version: string;
	};
}

export interface ClipboardStore {
	wakeUpKey: string;
	enableSound?: boolean;
	capacity?: string;
}
