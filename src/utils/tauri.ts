import {
	type ConfirmDialogOptions,
	ask as tauriAsk,
} from "@tauri-apps/api/dialog";
import { isObject } from "lodash-es";

export function ask(
	message: string,
	options?: string | ConfirmDialogOptions,
): Promise<boolean> {
	if (isMac() && isObject(options) && options.type === "warning") {
		options.type = "error";
	}
	return tauriAsk(message, options);
}
