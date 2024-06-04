export type Theme = "auto" | "light" | "dark";

export interface GlobalStore {
	theme: Theme;
	isDark: boolean;
	autostart: boolean;
	appInfo?: {
		name: string;
		version: string;
	};
}
