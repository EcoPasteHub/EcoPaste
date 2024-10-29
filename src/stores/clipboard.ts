import type { ClipboardStore } from "@/types/store";
import { proxy } from "valtio";

export const clipboardStore = proxy<ClipboardStore>({
	window: {
		style: "float",
		position: "remember",
		backTop: false,
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
	},

	history: {
		duration: 0,
		unit: 1,
	},
});
