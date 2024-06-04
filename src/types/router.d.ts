import type { WindowOptions } from "@tauri-apps/api/window";
import type { ComponentType } from "react";

export type Path = "/" | "/settings" | "/about";

export interface Route {
	path: Path;
	Component: ComponentType;
	children?: Route[];
	meta?: {
		icon?: string;
		title?: string;
		windowOptions?: WindowOptions;
	};
}
