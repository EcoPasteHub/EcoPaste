export type Theme = "auto" | "light" | "dark";

export type TabTrigger = "click" | "hover";

export interface GlobalStore {
	theme: Theme;
	isDark: boolean;
	autoStart: boolean;
	tabTrigger: TabTrigger;
	appInfo?: {
		name: string;
		version: string;
	};
}
