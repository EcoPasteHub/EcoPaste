import type { ClipboardStore } from "@/types/store";
import { proxy } from "valtio";

export const clipboardStore = proxy<ClipboardStore>({
	window: {
		style: "float",
		position: "remember",
		backTop: false,
		showAll: false,
	},

	audio: {
		copy: false,
	},

	search: {
		position: "top",
		defaultFocus: false,
		autoClear: false,
	},

	content: {
		autoPaste: "double",
		ocr: true,
		copyPlain: false,
		pastePlain: false,
		operationButtons: ["copy", "star", "delete"],
	},

	history: {
		duration: 0,
		unit: 1,
	},
});
