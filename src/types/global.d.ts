import "react";

declare module "react" {
	interface HTMLAttributes<T> extends AriaAttributes, DOMAttributes<T> {
		"data-tauri-drag-region"?: boolean;
	}
}

declare module "valtio" {
	function useSnapshot<T extends object>(p: T): T;
}
