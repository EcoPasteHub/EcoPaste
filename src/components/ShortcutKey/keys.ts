import { defaults } from "lodash-es";

export interface Key {
	key: string;
	symbol?: string;
	shortcut?: string;
	macosSymbol?: string;
}

export const modifierKeys: Key[] = [
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
		symbol: "Win",
		macosSymbol: "⌘",
	},
].map((item: Key) => ({
	...item,
	shortcut: item.key,
}));

export const normalKeys: Key[] = [
	// 第一排
	{
		key: "Escape",
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
		symbol: "-",
		shortcut: "-",
	},
	{
		key: "Equal",
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
		symbol: "[",
	},
	{
		key: "BracketRight",
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
		symbol: "⇡",
	},
	{
		key: "ArrowDown",
		symbol: "⇣",
	},
	{
		key: "ArrowLeft",
		symbol: "⇠",
	},
	{
		key: "ArrowRight",
		symbol: "⇢",
	},
].map((item: Key) => {
	const { key } = item;

	defaults(item, {
		symbol: key,
		shortcut: key,
	});

	if (key.startsWith("Digit") || key.startsWith("Key")) {
		item.shortcut = item.symbol = key.slice(-1);
	}

	item.macosSymbol = item.symbol;

	return item;
});

export const keys = modifierKeys.concat(normalKeys);
