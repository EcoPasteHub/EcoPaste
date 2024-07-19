import type { WindowOptions } from "@tauri-apps/api/window";
import type { RouteObject } from "react-router-dom";

export type RoutePath =
	| "/"
	| "/preference"
	| "clipboard"
	| "settings"
	| "about"
	| "data-backup";

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
