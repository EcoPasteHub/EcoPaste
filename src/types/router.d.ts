import type { WindowOptions } from "@tauri-apps/api/window";
import type { RouteObject } from "react-router-dom";

export type RoutePath =
	| "/"
	| "/preference"
	| "clipboard"
	| "general"
	| "about"
	| "data-backup"
	| "shortcut"
	| "history";

export interface Route extends RouteObject {
	path: RoutePath;
	Component: ComponentType;
	children?: Route[];
	meta?: {
		icon?: string;
		title?: string;
		windowOptions?: WindowOptions;
	};
}
