import { getName, getVersion } from "@tauri-apps/api/app";
import { emit } from "@tauri-apps/api/event";
import { appDataDir } from "@tauri-apps/api/path";
import {
  exists,
  mkdir,
  readTextFile,
  writeTextFile,
} from "@tauri-apps/plugin-fs";
import { platform } from "@tauri-apps/plugin-os";
import { omit } from "es-toolkit/compat";
import { getLocale } from "tauri-plugin-locale-api";
import { LISTEN_KEY } from "@/constants";
import { clipboardStore } from "@/stores/clipboard";
import { globalStore } from "@/stores/global";
import type { Language, Store } from "@/types/store";
import { normalizeAppearanceGroupTabs } from "@/utils/group";
import { deepAssign } from "./object";
import { getSaveStorePath } from "./path";

/**
 * 初始化配置项
 */
const initStore = async () => {
  try {
    globalStore.appearance.language ??= await getLocale<Language>();
    globalStore.env.platform = platform();
    globalStore.env.appName = await getName();
    globalStore.env.appVersion = await getVersion();
    globalStore.env.saveDataDir ??= await appDataDir();
  } catch (e) {
    // 浏览器环境下提供默认值
    globalStore.appearance.language ??= "zh-CN";
    globalStore.env.platform = "browser" as any;
    globalStore.env.appName = "EcoPaste";
    globalStore.env.appVersion = "0.0.0";
    globalStore.env.saveDataDir ??= "";
  }

  // @ts-expect-error
  if (clipboardStore.window.style === "float") {
    clipboardStore.window.style = "standard";
  }

  globalStore.appearance.groupTabs = normalizeAppearanceGroupTabs(
    globalStore.appearance.groupTabs,
  );

  if (globalStore.env.saveDataDir) {
    try {
      await mkdir(globalStore.env.saveDataDir, { recursive: true });
    } catch (e) {
      // 忽略
    }
  }
};

/**
 * 本地存储配置项
 * @param backup 是否为备份数据
 */
export const saveStore = async (backup = false) => {
  const store = { clipboardStore, globalStore };

  const path = await getSaveStorePath(backup);

  return writeTextFile(path, JSON.stringify(store, null, 2));
};

export const syncStore = async () => {
  await emit(LISTEN_KEY.STORE_CHANGED, { clipboardStore, globalStore });

  return saveStore();
};

/**
 * 从本地存储恢复配置项
 * @param backup 是否为备份数据
 */
export const restoreStore = async (backup = false) => {
  const path = await getSaveStorePath(backup);

  const existed = await exists(path);

  if (existed) {
    const content = await readTextFile(path);
    const store: Store = JSON.parse(content);
    const nextGlobalStore = omit(store.globalStore, backup ? "env" : "");

    deepAssign(globalStore, nextGlobalStore);
    deepAssign(clipboardStore, store.clipboardStore);
  }

  if (backup) return;

  return initStore();
};
