import type { WindowOptions } from "@tauri-apps/api/window";
import type { ComponentType } from "react";

export interface Route {
	path: "/" | "/settings" | "/about" | "/clipboard-history" | "/data-backup";
	Component: ComponentType;
	children?: Route[];
	meta?: {
		icon?: string;
		title?: string;
		windowOptions?: WindowOptions;
	};
}
