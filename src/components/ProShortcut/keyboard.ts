import { defaults } from "es-toolkit/compat";

export interface Key {
  eventKey: string;
  hookKey?: string;
  tauriKey?: string;
  symbol?: string;
}

export const modifierKeys: Key[] = [
  {
    eventKey: "Shift",
    symbol: isMac ? "⇧" : "Shift",
  },
  {
    eventKey: "Control",
    hookKey: "ctrl",
    symbol: isMac ? "⌃" : "Ctrl",
  },
  {
    eventKey: "Alt",
    symbol: isMac ? "⌥" : "Alt",
  },
  {
    eventKey: "Command",
    hookKey: "meta",
    symbol: isMac ? "⌘" : "Super",
  },
].map((item) => {
  const { eventKey } = item;

  defaults<Key, Partial<Key>>(item, {
    hookKey: eventKey.toLowerCase(),
    tauriKey: eventKey,
  });

  return item;
});

export const standardKeys: Key[] = [
  // 第一排
  {
    eventKey: "Escape",
    hookKey: "esc",
    symbol: isMac ? "⎋" : "Esc",
  },
  {
    eventKey: "F1",
  },
  {
    eventKey: "F2",
  },
  {
    eventKey: "F3",
  },
  {
    eventKey: "F4",
  },
  {
    eventKey: "F5",
  },
  {
    eventKey: "F6",
  },
  {
    eventKey: "F7",
  },
  {
    eventKey: "F8",
  },
  {
    eventKey: "F9",
  },
  {
    eventKey: "F10",
  },
  {
    eventKey: "F11",
  },
  {
    eventKey: "F12",
  }, // 第二排
  {
    eventKey: "Backquote",
    hookKey: "graveaccent",
    symbol: "`",
  },
  {
    eventKey: "Digit1",
  },
  {
    eventKey: "Digit2",
  },
  {
    eventKey: "Digit3",
  },
  {
    eventKey: "Digit4",
  },
  {
    eventKey: "Digit5",
  },
  {
    eventKey: "Digit6",
  },
  {
    eventKey: "Digit7",
  },
  {
    eventKey: "Digit8",
  },
  {
    eventKey: "Digit9",
  },
  {
    eventKey: "Digit0",
  },
  {
    eventKey: "Minus",
    hookKey: "dash",
    symbol: "-",
    tauriKey: "-",
  },
  {
    eventKey: "Equal",
    hookKey: "equalsign",
    symbol: "=",
    tauriKey: "=",
  },
  {
    eventKey: "Backspace",
    symbol: isMac ? "⌫" : void 0,
  },
  // 第三排
  {
    eventKey: "Tab",
    symbol: isMac ? "⇥" : void 0,
  },
  {
    eventKey: "KeyQ",
  },
  {
    eventKey: "KeyW",
  },
  {
    eventKey: "KeyE",
  },
  {
    eventKey: "KeyR",
  },
  {
    eventKey: "KeyT",
  },
  {
    eventKey: "KeyY",
  },
  {
    eventKey: "KeyU",
  },
  {
    eventKey: "KeyI",
  },
  {
    eventKey: "KeyO",
  },
  {
    eventKey: "KeyP",
  },
  {
    eventKey: "BracketLeft",
    hookKey: "openbracket",
    symbol: "[",
  },
  {
    eventKey: "BracketRight",
    hookKey: "closebracket",
    symbol: "]",
  },
  {
    eventKey: "Backslash",
    symbol: "\\",
  },
  // 第四排
  {
    eventKey: "KeyA",
  },
  {
    eventKey: "KeyS",
  },
  {
    eventKey: "KeyD",
  },
  {
    eventKey: "KeyF",
  },
  {
    eventKey: "KeyG",
  },
  {
    eventKey: "KeyH",
  },
  {
    eventKey: "KeyJ",
  },
  {
    eventKey: "KeyK",
  },
  {
    eventKey: "KeyL",
  },
  {
    eventKey: "Semicolon",
    symbol: ";",
  },
  {
    eventKey: "Quote",
    hookKey: "singlequote",
    symbol: "'",
  },
  {
    eventKey: "Enter",
    symbol: isMac ? "↩︎" : void 0,
  },
  // 第五排
  {
    eventKey: "KeyZ",
  },
  {
    eventKey: "KeyX",
  },
  {
    eventKey: "KeyC",
  },
  {
    eventKey: "KeyV",
  },
  {
    eventKey: "KeyB",
  },
  {
    eventKey: "KeyN",
  },
  {
    eventKey: "KeyM",
  },
  {
    eventKey: "Comma",
    symbol: ",",
  },
  {
    eventKey: "Period",
    symbol: ".",
  },
  {
    eventKey: "Slash",
    hookKey: "forwardslash",
    symbol: "/",
  },
  // 第六排
  {
    eventKey: "Space",
    symbol: isMac ? "␣" : void 0,
  },
  // 方向键
  {
    eventKey: "ArrowUp",
    hookKey: "uparrow",
    symbol: "↑",
  },
  {
    eventKey: "ArrowDown",
    hookKey: "downarrow",
    symbol: "↓",
  },
  {
    eventKey: "ArrowLeft",
    hookKey: "leftarrow",
    symbol: "←",
  },
  {
    eventKey: "ArrowRight",
    hookKey: "rightarrow",
    symbol: "→",
  },
  // 功能键
  {
    eventKey: "Delete",
    symbol: isMac ? "⌫" : void 0,
  },
].map((item) => {
  const { eventKey } = item;

  defaults<Key, Partial<Key>>(item, {
    hookKey: eventKey.toLowerCase(),
    symbol: eventKey,
    tauriKey: eventKey,
  });

  if (eventKey.startsWith("Digit") || eventKey.startsWith("Key")) {
    item.tauriKey = item.symbol = eventKey.slice(-1);

    item.hookKey = item.tauriKey.toLowerCase();
  }

  return item;
});

export const keys = modifierKeys.concat(standardKeys);

export const getKeySymbol = (key: string) => {
  const fields = ["tauriKey", "hookKey"] as const;

  const matched = keys.find((entry) => {
    return fields.some((field) => entry[field] === key);
  });

  return matched?.symbol ?? key;
};
