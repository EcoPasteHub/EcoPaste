import { useMount } from "ahooks";
import { cloneDeep } from "es-toolkit";
import { isEmpty, remove } from "es-toolkit/compat";
import { nanoid } from "nanoid";
import {
  type ClipboardChangeOptions,
  onClipboardChange,
  startListening,
} from "tauri-plugin-clipboard-x-api";
import { fullName } from "tauri-plugin-fs-pro-api";
import {
  insertHistory,
  selectHistory,
  updateHistory,
} from "@/database/history";
import type { State } from "@/pages/Main";
import { getClipboardTextSubtype } from "@/plugins/clipboard";
import { clipboardStore } from "@/stores/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";
import { formatDate } from "@/utils/dayjs";

export const useClipboard = (
  state: State,
  options?: ClipboardChangeOptions,
) => {
  useMount(async () => {
    await startListening();

    onClipboardChange(async (result) => {
      const { files, image, html, rtf, text } = result;

      if (isEmpty(result) || Object.values(result).every(isEmpty)) return;

      const { copyPlain } = clipboardStore.content;

      const data = {
        createTime: formatDate(),
        favorite: false,
        group: "text",
        id: nanoid(),
        search: text?.value,
      } as DatabaseSchemaHistory;

      if (files) {
        Object.assign(data, files, {
          group: "files",
          search: files.value.join(" "),
        });
      } else if (html && !copyPlain) {
        Object.assign(data, html);
      } else if (rtf && !copyPlain) {
        Object.assign(data, rtf);
      } else if (text) {
        const subtype = await getClipboardTextSubtype(text.value);

        Object.assign(data, text, {
          subtype,
        });
      } else if (image) {
        Object.assign(data, image, {
          group: "image",
        });
      }

      const sqlData = cloneDeep(data);

      const { type, value, group, createTime } = data;

      if (type === "image") {
        sqlData.value = await fullName(value);
      }

      if (type === "files") {
        sqlData.value = JSON.stringify(value);
      }

      const [matched] = await selectHistory((qb) => {
        const { type, value } = sqlData;

        return qb.where("type", "=", type).where("value", "=", value);
      });

      const visible = state.group === "all" || state.group === group;

      if (matched) {
        if (!clipboardStore.content.autoSort) return;

        const { id } = matched;

        if (visible) {
          remove(state.list, { id });

          state.list.unshift({ ...data, id });
        }

        return updateHistory(id, { createTime });
      }

      if (visible) {
        state.list.unshift(data);
      }

      insertHistory(sqlData);
    }, options);
  });
};
