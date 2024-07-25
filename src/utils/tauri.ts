import {
	type ConfirmDialogOptions,
	ask as tauriAsk,
} from "@tauri-apps/api/dialog";

export function ask(
	message: string,
	options?: string | ConfirmDialogOptions,
): Promise<boolean> {
	if (isMac() && typeof options === "object" && options.type === "warning") {
		options.type = "error";
	}
	return tauriAsk(message, options);
}
