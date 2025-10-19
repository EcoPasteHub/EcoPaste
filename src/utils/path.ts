import { getName } from "@tauri-apps/api/app";
import { appDataDir, sep } from "@tauri-apps/api/path";
import { last } from "es-toolkit";

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
  return join(globalStore.env.saveDataDir!);
};

/**
 * 获取数据库文件存储路径
 */
export const getSaveDatabasePath = async () => {
  const appName = await getName();
  const extname = isDev() ? "dev.db" : "db";

  return join(getSaveDataPath(), `${appName}.${extname}`);
};

/**
 * 获取存储图片的路径
 */
export const getSaveImagePath = () => {
  return join(getSaveDataPath(), "images");
};

/**
 * 解析完整的存储图片路径
 * @param file 文件名
 */
export const resolveImagePath = (file: string) => {
  if (file.startsWith(getSaveImagePath())) return file;

  return join(getSaveImagePath(), file);
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
