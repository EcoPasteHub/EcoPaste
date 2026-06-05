import { STORAGE_ERROR_BYTES, STORAGE_WARNING_BYTES } from "../constants";

interface StorageToneClass {
  bg: string;
  text: string;
}

/**
 * 把字节数格式化成侧栏里的紧凑存储标签。
 */
export function formatBytes(bytes: number) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }

  const digits = unitIndex === 0 || size >= 10 ? 0 : 1;

  return `${size.toFixed(digits)} ${units[unitIndex]}`;
}

/**
 * 用离散宽度表达本地数据总量所在档位，避免 inline style。
 */
export function storageMeterClass(totalBytes: number) {
  if (totalBytes <= 0) return "w-1/5";
  if (totalBytes >= STORAGE_ERROR_BYTES) return "w-full";
  if (totalBytes >= STORAGE_WARNING_BYTES) return "w-3/5";

  return "w-1/5";
}

/**
 * 返回当前总量对应的目标档位，用于展示“当前 / 目标”。
 */
export function storageTargetBytes(totalBytes: number) {
  if (totalBytes < STORAGE_WARNING_BYTES) return STORAGE_WARNING_BYTES;

  return STORAGE_ERROR_BYTES;
}

/**
 * 按本地数据总量返回状态色：低占用绿色、中等黄色、高占用红色。
 */
export function storageToneClass(totalBytes: number): StorageToneClass {
  if (totalBytes >= STORAGE_ERROR_BYTES) {
    return { bg: "bg-ant-error", text: "text-ant-error" };
  }

  if (totalBytes >= STORAGE_WARNING_BYTES) {
    return { bg: "bg-ant-warning", text: "text-ant-warning" };
  }

  return { bg: "bg-ant-success", text: "text-ant-success" };
}
