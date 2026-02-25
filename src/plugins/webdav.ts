import { invoke } from "@tauri-apps/api/core";

export interface WebdavConfig {
  address: string;
  username: string;
  password: string;
  path: string;
}

export interface WebdavBackupFile {
  fileName: string;
  size?: number;
  modified?: string;
}

const COMMAND = {
  CANCEL_UPLOAD: "plugin:eco-webdav|cancel_upload",
  CREATE_SLIM_DATABASE: "plugin:eco-webdav|create_slim_database",
  DELETE_BACKUP: "plugin:eco-webdav|delete_backup",
  DOWNLOAD_BACKUP: "plugin:eco-webdav|download_backup",
  GET_COMPUTER_NAME: "plugin:eco-webdav|get_computer_name",
  GET_CONFIG: "plugin:eco-webdav|get_config",
  LIST_BACKUPS: "plugin:eco-webdav|list_backups",
  SET_CONFIG: "plugin:eco-webdav|set_config",
  TEST_CONFIG: "plugin:eco-webdav|test_config",
  UPLOAD_BACKUP: "plugin:eco-webdav|upload_backup",
};

export const setWebdavConfig = (config: WebdavConfig) => {
  return invoke<void>(COMMAND.SET_CONFIG, { config });
};

export const getWebdavConfig = () => {
  return invoke<WebdavConfig | null>(COMMAND.GET_CONFIG);
};

export const getWebdavComputerName = () => {
  return invoke<string>(COMMAND.GET_COMPUTER_NAME);
};

export const testWebdavConfig = (config: WebdavConfig) => {
  return invoke<void>(COMMAND.TEST_CONFIG, { config });
};

export const listWebdavBackups = async () => {
  const list = await invoke<
    { file_name: string; size?: number; modified?: string }[]
  >(COMMAND.LIST_BACKUPS);

  return list.map((item) => ({
    fileName: item.file_name,
    modified: item.modified,
    size: item.size,
  })) as WebdavBackupFile[];
};

export const uploadWebdavBackup = (filePath: string, fileName: string) => {
  return invoke<void>(COMMAND.UPLOAD_BACKUP, {
    fileName,
    filePath,
  });
};

export const cancelWebdavUpload = () => {
  return invoke<void>(COMMAND.CANCEL_UPLOAD);
};

export const downloadWebdavBackup = (fileName: string) => {
  return invoke<string>(COMMAND.DOWNLOAD_BACKUP, { fileName });
};

export const deleteWebdavBackup = (fileName: string) => {
  return invoke<void>(COMMAND.DELETE_BACKUP, { fileName });
};

export const createSlimDatabase = (
  sourceDbPath: string,
  targetDbPath: string,
) => {
  return invoke<void>(COMMAND.CREATE_SLIM_DATABASE, {
    sourceDbPath,
    targetDbPath,
  });
};
