import { getName } from "@tauri-apps/api/app";
import { appDataDir, sep } from "@tauri-apps/api/path";
import { last } from "es-toolkit";
import { globalStore } from "@/stores/global";
import { isDev } from "./is";

/**
 * 拼接文件路径
 * @param paths 路径数组
 */
export function join(...paths: string[]) {
  const joinPaths = paths.map((path, index) => {
    if (index === 0) {
      return path.replace(new RegExp(`${sep()}+$`), "");
    }

    return path.replace(new RegExp(`^${sep()}+|${sep()}+$`, "g"), "");
  });

  return joinPaths.join(sep());
}

/**
 * 获取存储数据的目录
 */
export const getSaveDataPath = () => {
  // 开发环境可强制指向测试数据目录，避免读写真实 AppData
  const testDataDir = isDev() ? import.meta.env.VITE_ECOPASTE_TESTDATA_DIR : "";

  if (testDataDir) {
    return join(testDataDir);
  }

  return join(globalStore.env.saveDataDir!);
};

/**
 * 获取数据库文件存储路径
 */
export const getSaveDatabasePath = async () => {
  const appName = await getName();
  // 开发环境下若指定了测试数据目录，则直接复用生产库文件名 `*.db`
  // 这样可以用真实大库验证搜索性能（同时不会影响真实数据，因为目录已是拷贝）。
  const testDataDir = isDev() ? import.meta.env.VITE_ECOPASTE_TESTDATA_DIR : "";
  const extname = testDataDir ? "db" : isDev() ? "dev.db" : "db";

  return join(getSaveDataPath(), `${appName}.${extname}`);
};

/**
 * 获取存储图片的路径
 */
export const getSaveImagePath = () => {
  return join(getSaveDataPath(), "images");
};

/**
 * 存储数据的目录名
 */
export const getSaveDataDirName = () => {
  return last(getSaveDataPath().split(sep())) as string;
};

/**
 * 存储配置项的路径
 * @param backup 是否是备份数据
 */
export const getSaveStorePath = async (backup = false) => {
  const extname = isDev() ? "dev.json" : "json";

  if (backup) {
    return join(getSaveDataPath(), `.store-backup.${extname}`);
  }

  return join(await appDataDir(), `.store.${extname}`);
};

/**
 * 存储窗口位置的路径
 */
export const getSaveWindowStatePath = async () => {
  const extname = isDev() ? "dev.json" : "json";

  return join(await appDataDir(), `.window-state.${extname}`);
};
