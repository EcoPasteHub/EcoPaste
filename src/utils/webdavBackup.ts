import { emit } from "@tauri-apps/api/event";
import { tempDir } from "@tauri-apps/api/path";
import { remove } from "@tauri-apps/plugin-fs";
import { platform } from "@tauri-apps/plugin-os";
import { compress, decompress, fullName } from "tauri-plugin-fs-pro-api";
import { LISTEN_KEY } from "@/constants";
import {
  createSlimDatabase,
  downloadWebdavBackup,
  getWebdavComputerName,
  listWebdavBackups,
  uploadWebdavBackup,
} from "@/plugins/webdav";
import { globalStore } from "@/stores/global";
import { dayjs, formatDate } from "@/utils/dayjs";
import {
  getBackupExtname,
  getSaveDatabasePath,
  getSaveDataPath,
  getSaveImagePath,
  getSaveStorePath,
  join,
} from "@/utils/path";
import { wait } from "@/utils/shared";
import { restoreStore, saveStore } from "@/utils/store";

let cachedComputerName: string | undefined;

export const buildBackupTime = () => {
  return formatDate(dayjs(), "YYYYMMDDHHmmss");
};

const normalizeComputerName = (value?: string) => {
  const name = value?.trim();
  if (!name) return "Unknown";
  return name;
};

const resolveComputerName = async (value?: string) => {
  if (value?.trim()) {
    cachedComputerName = value.trim();
    return cachedComputerName;
  }
  if (cachedComputerName) return cachedComputerName;
  const fetched = await getWebdavComputerName();
  cachedComputerName = normalizeComputerName(fetched);
  return cachedComputerName;
};

const capitalizeFirst = (value: string) => {
  if (!value) return value;
  return value.slice(0, 1).toUpperCase() + value.slice(1);
};

export const getDefaultWebdavFilename = async (computerName?: string) => {
  const appName = globalStore.env.appName || "EcoPaste";
  const device = await resolveComputerName(computerName);
  const deviceName = normalizeComputerName(device).replace(/\s+/g, "");
  const os = capitalizeFirst(await platform());
  const timestamp = buildBackupTime();
  return `${appName}.${timestamp}.${deviceName}.${os}`;
};

export const normalizeWebdavBackupFileName = (fileName: string) => {
  const extname = getBackupExtname();
  if (fileName.endsWith(`.${extname}`)) return fileName;
  if (fileName.toLowerCase().endsWith(".zip")) {
    return `${fileName.slice(0, -4)}.${extname}`;
  }
  return `${fileName}.${extname}`;
};

export const getDefaultWebdavBackupFileName = async (computerName?: string) => {
  return normalizeWebdavBackupFileName(
    await getDefaultWebdavFilename(computerName),
  );
};

export const listWebdavBackupFiles = async () => {
  const list = await listWebdavBackups();
  const extname = getBackupExtname();
  return list.filter((item) => item.fileName.endsWith(`.${extname}`));
};

export const createWebdavBackupArchive = async (
  fileName: string,
  slim: boolean,
) => {
  await saveStore(true);

  const basePath = getSaveDataPath();
  const tempRoot = await tempDir();
  const archivePath = join(tempRoot, fileName);
  const includes: string[] = [await fullName(await getSaveStorePath(true))];
  const cleanupPaths: string[] = [];

  if (slim) {
    const sourceDbPath = await getSaveDatabasePath();
    const slimDbPath = join(basePath, ".slim-backup.db");
    await createSlimDatabase(sourceDbPath, slimDbPath);
    cleanupPaths.push(slimDbPath);
    includes.push(await fullName(slimDbPath));
  } else {
    includes.push(
      await fullName(getSaveImagePath()),
      await fullName(await getSaveDatabasePath()),
    );
  }

  await compress(basePath, archivePath, { includes });

  return {
    archivePath,
    cleanupPaths,
  };
};

export const cleanupBackupFiles = async (paths: string[]) => {
  for (const path of paths) {
    try {
      await remove(path);
    } catch (error) {
      void error;
    }
  }
};

export const backupToWebdav = async (fileName: string, slim: boolean) => {
  const resolvedName = normalizeWebdavBackupFileName(fileName);
  const { archivePath, cleanupPaths } = await createWebdavBackupArchive(
    resolvedName,
    slim,
  );
  try {
    await uploadWebdavBackup(archivePath, resolvedName);
  } finally {
    await cleanupBackupFiles([archivePath, ...cleanupPaths]);
  }
};

export const restoreWebdavBackup = async (fileName: string) => {
  const path = await downloadWebdavBackup(fileName);
  emit(LISTEN_KEY.CLOSE_DATABASE);
  await wait();
  await decompress(path, getSaveDataPath());
  await remove(path);
  await restoreStore(true);
  emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);
};
