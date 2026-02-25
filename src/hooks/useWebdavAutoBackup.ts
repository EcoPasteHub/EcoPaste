import { emit } from "@tauri-apps/api/event";
import { useUpdateEffect } from "ahooks";
import { useEffect, useRef } from "react";
import { useSnapshot } from "valtio";
import { LISTEN_KEY } from "@/constants";
import { deleteWebdavBackup } from "@/plugins/webdav";
import { clipboardStore } from "@/stores/clipboard";
import { formatDate } from "@/utils/dayjs";
import {
  backupToWebdav,
  getDefaultWebdavBackupFileName,
  listWebdavBackupFiles,
} from "@/utils/webdavBackup";

export const useWebdavAutoBackup = () => {
  const { webdav } = useSnapshot(clipboardStore);
  const timerRef = useRef<ReturnType<typeof setInterval>>();

  const updateStatus = (status: "success" | "error", error?: string) => {
    clipboardStore.webdav.lastBackupStatus = status;
    clipboardStore.webdav.lastBackupAt = formatDate();
    clipboardStore.webdav.lastBackupError = error;
  };

  const trimBackups = async (maxBackups: number) => {
    if (maxBackups <= 0) return;
    const list = await listWebdavBackupFiles();
    const sorted = list
      .map((item) => ({
        ...item,
        timeValue: parseTimeValue(item.fileName) || 0,
      }))
      .sort((a, b) => b.timeValue - a.timeValue);
    const excess = sorted.slice(maxBackups);
    if (excess.length === 0) return;
    await Promise.all(excess.map((item) => deleteWebdavBackup(item.fileName)));
  };

  const parseTimeValue = (fileName: string) => {
    const parts = fileName.split(".");
    const timestamp = parts[1];
    if (!timestamp || timestamp.length !== 14) return void 0;
    const value = Number(timestamp);
    if (Number.isNaN(value)) return void 0;
    return value;
  };

  const run = async () => {
    try {
      const fileName = await getDefaultWebdavBackupFileName();
      await backupToWebdav(fileName, webdav.slim);
      await trimBackups(webdav.maxBackups);
      updateStatus("success");
    } catch (error: any) {
      updateStatus("error", String(error));
    } finally {
      emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);
    }
  };

  useEffect(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
    }

    if (!webdav.autoBackup) return;

    const delay = webdav.autoBackup * 60 * 1000;

    timerRef.current = setInterval(run, delay);

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [webdav.autoBackup, webdav.maxBackups, webdav.slim]);

  useUpdateEffect(() => {
    if (webdav.autoBackup) {
      run();
    }
  }, [webdav.autoBackup]);
};
