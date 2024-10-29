import type { GlobalStore } from "@/types/store";
import { proxy } from "valtio";
import { subscribeKey as valtioSubscribeKey } from "valtio/utils";

export const globalStore = proxy<GlobalStore>({
	app: {
		autoStart: false,
		silentStart: false,
		showMenubarIcon: true,
		showTaskbarIcon: false,
	},

	appearance: {
		theme: "auto",
		isDark: false,
	},

	update: {
		auto: false,
		beta: false,
	},

	shortcut: {
		clipboard: "Alt+C",
		preference: "Alt+X",
		quickPaste: {
			enable: false,
			value: "Command+Shift",
		},
	},

	env: {},
});

export const subscribeKey: typeof valtioSubscribeKey = (
	object,
	key,
	callback,
	immediate,
) => {
	if (immediate) {
		callback(object[key]);
	}

	return valtioSubscribeKey(object, key, callback);
};
