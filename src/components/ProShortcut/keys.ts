import { defaults, find } from "lodash-es";

// TODO: 添加一个主修饰键，就不用导出 isMac 了
// TODO:  快捷键 可以写到 constants 中

export interface Key {
	key: string;
	symbol?: string;
	shortcut?: string;
	webShortcut?: string;
}

export const modifierKeys: Key[] = [
	{
		key: "Shift",
		symbol: isMac ? "⇧" : "Shift",
	},
	{
		key: "Control",
		symbol: isMac ? "⌃" : "Ctrl",
	},
	{
		key: "Alt",
		symbol: isMac ? "⌥" : "Alt",
	},
	{
		key: "Command",
		symbol: isMac ? "⌘" : "Win",
		webShortcut: "meta",
	},
].map((item) => {
	const { key, symbol } = item;

	return {
		shortcut: key,
		webShortcut: symbol?.toLowerCase(),
		...item,
	};
});

export const normalKeys: Key[] = [
	// 第一排
	{
		key: "Escape",
		symbol: isMac ? "⎋" : "Esc",
		webShortcut: "esc",
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
		symbol: "`",
		webShortcut: "graveaccent",
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
		symbol: "-",
		shortcut: "-",
		webShortcut: "dash",
	},
	{
		key: "Equal",
		symbol: "=",
		shortcut: "=",
		webShortcut: "equalsign",
	},
	{
		key: "Backspace",
		symbol: isMac ? "⌫" : "Backspace",
	},
	// 第三排
	{
		key: "Tab",
		symbol: isMac ? "⇥" : "Tab",
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
		symbol: "[",
		webShortcut: "openbracket",
	},
	{
		key: "BracketRight",
		symbol: "]",
		webShortcut: "closebracket",
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
		symbol: "'",
		webShortcut: "singlequote",
	},
	{
		key: "Enter",
		symbol: isMac ? "↩︎" : "Enter",
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
		symbol: "/",
		webShortcut: "forwardslash",
	},
	// 第六排
	{
		key: "Space",
		symbol: isMac ? "␣" : "Space",
	},
	// 方向键
	{
		key: "ArrowUp",
		symbol: "↑",
		webShortcut: "uparrow",
	},
	{
		key: "ArrowDown",
		symbol: "↓",
		webShortcut: "downarrow",
	},
	{
		key: "ArrowLeft",
		symbol: "←",
		webShortcut: "leftarrow",
	},
	{
		key: "ArrowRight",
		symbol: "→",
		webShortcut: "rightarrow",
	},
	// 功能键
	{
		key: "Delete",
		symbol: isMac ? "⌫" : "Del",
	},
].map((item) => {
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

	return item;
});

export const keys = modifierKeys.concat(normalKeys);

export const getKeySymbol = (key: string) => {
	return find(keys, { key })?.symbol || key;
};

export const getWebShortcuts = (shortcuts?: string) => {
	if (!shortcuts) return "";

	return shortcuts
		.split("+")
		.map((shortcut) => find(keys, { shortcut })?.webShortcut)
		.join(".");
};
