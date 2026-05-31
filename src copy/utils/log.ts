// 前端统一通过 tauri-plugin-log 走到 Rust 日志（同时进入 LogDir 文件 + Stdout + Webview console），
// 与 Rust 侧 `log::error!` 同源，避免分散在 console 里只在 devtools 才能看到。
// 注意：plugin-log 的 IPC 是 Promise；调用方不需要 await，但要 catch 兜底，避免日志失败再触发未捕获 promise 报错。

import {
  debug as pluginDebug,
  error as pluginError,
  info as pluginInfo,
  warn as pluginWarn,
} from "@tauri-apps/plugin-log";

type Payload = unknown;

function format(message: string, payload?: Payload): string {
  if (payload === undefined) return message;
  if (payload instanceof Error) {
    return `${message}: ${payload.stack ?? payload.message}`;
  }
  try {
    return `${message}: ${JSON.stringify(payload)}`;
  } catch {
    return `${message}: ${String(payload)}`;
  }
}

function safe(fn: (msg: string) => Promise<void>, msg: string): void {
  fn(msg).catch(() => {
    // 启动期 IPC 通道可能尚未就绪；落控制台兜底，避免 unhandled rejection。
    // biome-ignore lint/suspicious/noConsole: log channel fallback
    console.error("[log fallback]", msg);
  });
}

export const log = {
  debug: (message: string, payload?: Payload) =>
    safe(pluginDebug, format(message, payload)),
  error: (message: string, payload?: Payload) =>
    safe(pluginError, format(message, payload)),
  info: (message: string, payload?: Payload) =>
    safe(pluginInfo, format(message, payload)),
  warn: (message: string, payload?: Payload) =>
    safe(pluginWarn, format(message, payload)),
};
