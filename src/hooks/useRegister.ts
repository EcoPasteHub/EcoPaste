import {
  isRegistered,
  register,
  type ShortcutHandler,
  unregister,
} from "@tauri-apps/plugin-global-shortcut";
import { useAsyncEffect, useUnmount } from "ahooks";
import { castArray } from "es-toolkit/compat";
import { useState } from "react";

export const useRegister = (
  handler: ShortcutHandler,
  deps: Array<string | string[] | undefined>,
) => {
  const [oldShortcuts, setOldShortcuts] = useState(deps[0]);

  useAsyncEffect(async () => {
    const [shortcuts] = deps;

    for await (const shortcut of castArray(oldShortcuts)) {
      if (!shortcut) continue;

      const registered = await isRegistered(shortcut);

      if (registered) {
        await unregister(shortcut);
      }
    }

    if (!shortcuts) return;

    await register(shortcuts, (event) => {
      if (event.state === "Released") return;

      handler(event);
    });

    setOldShortcuts(shortcuts);
  }, deps);

  useUnmount(() => {
    const [shortcuts] = deps;

    if (!shortcuts) return;

    unregister(shortcuts);
  });
};
