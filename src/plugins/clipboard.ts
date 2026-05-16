import { exists } from "@tauri-apps/plugin-fs";
import {
  writeFiles,
  writeHTML,
  writeImage,
  writeRTF,
  writeText,
} from "tauri-plugin-clipboard-x-api";
import { clipboardStore } from "@/stores/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";
import { isColor, isEmail, isURL } from "@/utils/is";
import { paste } from "./paste";

const ignoredClipboardTextMap = new Map<string, number>();
const IGNORED_CLIPBOARD_TTL = 3000;

const cleanupIgnoredClipboardText = () => {
  const now = Date.now();

  ignoredClipboardTextMap.forEach((expiresAt, value) => {
    if (expiresAt <= now) {
      ignoredClipboardTextMap.delete(value);
    }
  });
};

export const ignoreNextClipboardTextHistory = (value: string) => {
  if (!value) return;

  cleanupIgnoredClipboardText();
  ignoredClipboardTextMap.set(value, Date.now() + IGNORED_CLIPBOARD_TTL);
};

export const shouldIgnoreClipboardTextHistory = (value?: string) => {
  if (!value) return false;

  cleanupIgnoredClipboardText();

  const expiresAt = ignoredClipboardTextMap.get(value);
  if (!expiresAt) return false;

  ignoredClipboardTextMap.delete(value);

  return expiresAt > Date.now();
};

export const getClipboardTextSubtype = async (value: string) => {
  try {
    if (isURL(value)) {
      return "url";
    }

    if (isEmail(value)) {
      return "email";
    }

    if (isColor(value)) {
      return "color";
    }

    if (await exists(value)) {
      return "path";
    }
  } catch {
    return;
  }
};

export const writeToClipboard = (data: DatabaseSchemaHistory) => {
  const { type, value, search } = data;

  switch (type) {
    case "text":
      return writeText(value);
    case "rtf":
      return writeRTF(search, value);
    case "html":
      return writeHTML(search, value);
    case "image":
      return writeImage(value);
    case "files":
      return writeFiles(value);
  }
};

export const copyTextToClipboard = (
  value: string,
  options?: { ignoreHistory?: boolean },
) => {
  if (options?.ignoreHistory) {
    ignoreNextClipboardTextHistory(value);
  }

  return writeText(value);
};

export const pasteTextToClipboard = async (
  value: string,
  options?: { ignoreHistory?: boolean },
) => {
  await copyTextToClipboard(value, options);

  return paste();
};

export const pasteToClipboard = async (
  data: DatabaseSchemaHistory,
  asPlain?: boolean,
) => {
  const { type, value, search } = data;
  const { pastePlain } = clipboardStore.content;

  if (asPlain ?? pastePlain) {
    if (type === "files") {
      await writeText(value.join("\n"));
    } else {
      await writeText(search);
    }
  } else {
    await writeToClipboard(data);
  }

  return paste();
};
