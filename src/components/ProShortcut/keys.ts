import { defaults, find, map } from "lodash-es";

export interface Key {
	key: string;
	symbol?: string;
	shortcut?: string;
	macosSymbol?: string;
	webShortcut?: string;
}

export const modifierKeys = map<Key, Key>(
	[
		{
			key: "Shift",
			symbol: "Shift",
			macosSymbol: "⇧",
		},
		{
			key: "Control",
			symbol: "Ctrl",
			macosSymbol: "⌃",
		},
		{
			key: "Alt",
			symbol: "Alt",
			macosSymbol: "⌥",
		},
		{
			key: "Command",
			webShortcut: "meta",
			symbol: "Win",
			macosSymbol: "⌘",
		},
	],
	(item) => {
		const { key, symbol } = item;

		return {
			shortcut: key,
			webShortcut: symbol?.toLowerCase(),
			...item,
		};
	},
);

export const normalKeys = map<Key, Key>(
	[
		// 第一排
		{
			key: "Escape",
			webShortcut: "esc",
			symbol: "⎋",
		},
		{
			key: "F1",
		},
		{
			key: "F2",
		},
		{
			key: "F3",
		},
		{
			key: "F4",
		},
		{
			key: "F5",
		},
		{
			key: "F6",
		},
		{
			key: "F7",
		},
		{
			key: "F8",
		},
		{
			key: "F9",
		},
		{
			key: "F10",
		},
		{
			key: "F11",
		},
		{
			key: "F12",
		}, // 第二排
		{
			key: "Backquote",
			webShortcut: "graveaccent",
			symbol: "`",
		},
		{
			key: "Digit1",
		},
		{
			key: "Digit2",
		},
		{
			key: "Digit3",
		},
		{
			key: "Digit4",
		},
		{
			key: "Digit5",
		},
		{
			key: "Digit6",
		},
		{
			key: "Digit7",
		},
		{
			key: "Digit8",
		},
		{
			key: "Digit9",
		},
		{
			key: "Digit0",
		},
		{
			key: "Minus",
			webShortcut: "dash",
			symbol: "-",
			shortcut: "-",
		},
		{
			key: "Equal",
			webShortcut: "equalsign",
			symbol: "=",
			shortcut: "=",
		},
		{
			key: "Backspace",
			symbol: "⌫",
		},
		// 第三排
		{
			key: "Tab",
			symbol: "⇥",
		},
		{
			key: "KeyQ",
		},
		{
			key: "KeyW",
		},
		{
			key: "KeyE",
		},
		{
			key: "KeyR",
		},
		{
			key: "KeyT",
		},
		{
			key: "KeyY",
		},
		{
			key: "KeyU",
		},
		{
			key: "KeyI",
		},
		{
			key: "KeyO",
		},
		{
			key: "KeyP",
		},
		{
			key: "BracketLeft",
			webShortcut: "openbracket",
			symbol: "[",
		},
		{
			key: "BracketRight",
			webShortcut: "closebracket",
			symbol: "]",
		},
		{
			key: "Backslash",
			symbol: "\\",
		},
		// 第四排
		{
			key: "KeyA",
		},
		{
			key: "KeyS",
		},
		{
			key: "KeyD",
		},
		{
			key: "KeyF",
		},
		{
			key: "KeyG",
		},
		{
			key: "KeyH",
		},
		{
			key: "KeyJ",
		},
		{
			key: "KeyK",
		},
		{
			key: "KeyL",
		},
		{
			key: "Semicolon",
			symbol: ";",
		},
		{
			key: "Quote",
			webShortcut: "singlequote",
			symbol: "'",
		},
		{
			key: "Enter",
			symbol: "↩︎",
		},
		// 第五排
		{
			key: "KeyZ",
		},
		{
			key: "KeyX",
		},
		{
			key: "KeyC",
		},
		{
			key: "KeyV",
		},
		{
			key: "KeyB",
		},
		{
			key: "KeyN",
		},
		{
			key: "KeyM",
		},
		{
			key: "Comma",
			symbol: ",",
		},
		{
			key: "Period",
			symbol: ".",
		},
		{
			key: "Slash",
			webShortcut: "forwardslash",
			symbol: "/",
		},
		// 第六排
		{
			key: "Space",
			symbol: "␣",
		},
		// 方向键
		{
			key: "ArrowUp",
			webShortcut: "uparrow",
			symbol: "⇡",
		},
		{
			key: "ArrowDown",
			webShortcut: "downarrow",
			symbol: "⇣",
		},
		{
			key: "ArrowLeft",
			webShortcut: "leftarrow",
			symbol: "⇠",
		},
		{
			key: "ArrowRight",
			webShortcut: "rightarrow",
			symbol: "⇢",
		},
	],
	(item) => {
		const { key } = item;

		defaults<Key, Partial<Key>>(item, {
			symbol: key,
			shortcut: key,
			webShortcut: key.toLowerCase(),
		});

		if (key.startsWith("Digit") || key.startsWith("Key")) {
			item.shortcut = item.symbol = key.slice(-1);

			item.webShortcut = item.shortcut.toLowerCase();
		}

		item.macosSymbol = item.symbol;

		return item;
	},
);

export const keys = modifierKeys.concat(normalKeys);

export const getWebShortcuts = (shortcuts?: string) => {
	if (!shortcuts) return "";

	return shortcuts
		.split("+")
		.map((shortcut) => find(keys, { shortcut })?.webShortcut)
		.join(".");
};
