import { exists } from "@tauri-apps/plugin-fs";
import {
  writeFiles,
  writeHTML,
  writeImage,
  writeRTF,
  writeText,
} from "tauri-plugin-clipboard-x-api";
import { getHistoryFullValue } from "@/database/history";
import { clipboardStore } from "@/stores/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";
import { isColor, isEmail, isURL } from "@/utils/is";
import { paste } from "./paste";

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

/**
 * 获取完整 value（列表里的 value 可能被截断，操作时需要完整内容）
 */
const resolveFullValue = async (data: DatabaseSchemaHistory) => {
  const { id, type, value } = data;

  // 图片类型的 value 是路径，不会被截断；files 类型在 useHistoryList 里已 parse
  if (type === "image" || type === "files") return value;

  // 文本类型：如果 value 看起来被截断了（等于 500 字符），从 DB 取完整值
  if (typeof value === "string" && value.length === 500) {
    return (await getHistoryFullValue(id)) ?? value;
  }

  return value;
};

export const writeToClipboard = async (data: DatabaseSchemaHistory) => {
  const { type, search } = data;
  const fullValue = await resolveFullValue(data);

  switch (type) {
    case "text":
      return writeText(fullValue);
    case "rtf":
      return writeRTF(search, fullValue);
    case "html":
      return writeHTML(search, fullValue);
    case "image":
      return writeImage(fullValue);
    case "files":
      return writeFiles(fullValue);
  }
};

export const pasteToClipboard = async (
  data: DatabaseSchemaHistory,
  asPlain?: boolean,
) => {
  const { type, search } = data;
  const { pastePlain } = clipboardStore.content;

  if (asPlain ?? pastePlain) {
    if (type === "files") {
      const fullValue = await resolveFullValue(data);

      await writeText(fullValue.join("\n"));
    } else {
      await writeText(search);
    }
  } else {
    await writeToClipboard(data);
  }

  return paste();
};
