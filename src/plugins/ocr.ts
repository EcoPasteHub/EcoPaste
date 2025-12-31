import { emit } from "@tauri-apps/api/event";
import { WebviewWindow } from "@tauri-apps/api/webviewWindow";
import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { error, info } from "@tauri-apps/plugin-log";
import { LISTEN_KEY, WINDOW_LABEL } from "@/constants";
import { globalStore } from "@/stores/global";

/**
 * 显示 OCR 窗口
 */
export const showOcrWindow = async () => {
  try {
    const ocrWindow = await WebviewWindow.getByLabel(WINDOW_LABEL.OCR);

    if (ocrWindow) {
      await ocrWindow.show();
      await ocrWindow.setFocus();
    } else {
      // 如果窗口不存在，创建新窗口
      const webview = new WebviewWindow(WINDOW_LABEL.OCR, {
        alwaysOnTop: true,
        center: true,
        decorations: false,
        height: 500,
        minHeight: 300,
        minWidth: 400,
        resizable: true,
        skipTaskbar: true,
        title: "OCR",
        transparent: true,
        url: "index.html/#/ocr",
        width: 450,
      });

      webview.once("tauri://created", () => {
        info("OCR window created");
      });
    }
  } catch (err) {
    error(`Failed to show OCR window: ${err}`);
  }
};

/**
 * 隐藏 OCR 窗口
 */
export const hideOcrWindow = async () => {
  try {
    const ocrWindow = await WebviewWindow.getByLabel(WINDOW_LABEL.OCR);

    if (ocrWindow) {
      await ocrWindow.hide();
    }
  } catch (err) {
    error(`Failed to hide OCR window: ${err}`);
  }
};

/**
 * 从文件选择图片并进行 OCR
 */
export const selectImageForOcr = async (withTranslate: boolean) => {
  const selected = await open({
    filters: [
      {
        extensions: ["png", "jpg", "jpeg", "gif", "bmp", "webp"],
        name: "Images",
      },
    ],
    multiple: false,
  });

  if (selected) {
    const filePath = typeof selected === "string" ? selected : selected;
    await processImageFile(filePath, withTranslate);
  }
};

/**
 * 处理图片文件
 */
export const processImageFile = async (
  filePath: string,
  withTranslate: boolean,
) => {
  try {
    // 读取文件为 base64
    const fileData = await readFile(filePath);
    const base64 = arrayBufferToBase64(fileData);

    // 显示 OCR 窗口
    await showOcrWindow();

    // 发送 OCR 开始事件
    await emit(LISTEN_KEY.OCR_START, {
      imageBase64: base64,
      withTranslate,
    });
  } catch (err) {
    error(`Failed to process image: ${err}`);
  }
};

/**
 * 从剪贴板获取图片并进行 OCR
 */
export const ocrFromClipboard = async (withTranslate: boolean) => {
  try {
    // 使用 clipboard-x 插件读取剪贴板图片
    const { readImage } = await import("tauri-plugin-clipboard-x-api");
    const imageBase64 = await readImage();

    if (imageBase64) {
      // 显示 OCR 窗口
      await showOcrWindow();

      // 发送 OCR 开始事件
      await emit(LISTEN_KEY.OCR_START, {
        imageBase64,
        withTranslate,
      });
    }
  } catch (err) {
    error(`Failed to read clipboard image: ${err}`);
  }
};

/**
 * 触发系统截图工具
 * Windows: Win + Shift + S
 * macOS: Cmd + Shift + 4
 * Linux: 使用 gnome-screenshot 或其他工具
 */
export const triggerScreenshot = async (withTranslate: boolean) => {
  // 隐藏主窗口（如果设置了）
  if (globalStore.ocr.hideMainWindow) {
    const { hideWindow } = await import("./window");
    await hideWindow();
  }

  // 延时等待用户完成截图
  await new Promise((resolve) => setTimeout(resolve, 500));

  // 从剪贴板读取图片
  await ocrFromClipboard(withTranslate);
};

/**
 * 将 ArrayBuffer 转换为 Base64
 */
const arrayBufferToBase64 = (buffer: Uint8Array): string => {
  let binary = "";
  const bytes = new Uint8Array(buffer);
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
};
