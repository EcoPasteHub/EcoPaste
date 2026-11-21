/**
 * 前端日志统一入口。
 *
 * - 始终通过 `tauri-plugin-log` 走到 Rust，落 LogDir 文件 + Stdout，与 Rust `log::error!` 同源。
 * - dev 环境额外打到浏览器 console（Rust 侧的 Webview target 也仅在 dev 启用，避免回灌到生产 webview）。
 * - plugin-log 的 IPC 是 Promise；调用方不需要 await，但要 catch 兜底，避免日志失败再触发未捕获 promise 报错。
 */

import {
  debug as pluginDebug,
  error as pluginError,
  info as pluginInfo,
  warn as pluginWarn,
} from "@tauri-apps/plugin-log";

import { isDev } from "./is";

type Payload = unknown;
type Level = "debug" | "info" | "warn" | "error";

/**
 * 把 message + 可选 payload 序列化成单行字符串，便于落文件检索。
 */
function format(message: string, payload?: Payload): string {
  if (payload === void 0) return message;

  if (payload instanceof Error) {
    return `${message}: ${payload.stack ?? payload.message}`;
  }

  try {
    return `${message}: ${JSON.stringify(payload)}`;
  } catch {
    return `${message}: ${String(payload)}`;
  }
}

/**
 * dev 环境下把日志同步打到浏览器 console，方便调试时直接看；
 * 生产环境跳过，避免无谓 IO（文件日志由 plugin-log 负责）。
 */
function devConsole(level: Level, message: string, payload?: Payload): void {
  if (!isDev) return;

  const args: unknown[] = payload === void 0 ? [message] : [message, payload];

  // biome-ignore lint/suspicious/noConsole: dev-only console mirror for log channel
  console[level](...args);
}

async function safe(
  fn: (msg: string) => Promise<void>,
  msg: string,
): Promise<void> {
  try {
    await fn(msg);
  } catch {
    // 启动期 IPC 通道可能尚未就绪；落控制台兜底，避免 unhandled rejection。
    // biome-ignore lint/suspicious/noConsole: log channel fallback
    console.error("[log fallback]", msg);
  }
}

export const log = {
  debug: (message: string, payload?: Payload) => {
    devConsole("debug", message, payload);
    safe(pluginDebug, format(message, payload));
  },
  error: (message: string, payload?: Payload) => {
    devConsole("error", message, payload);
    safe(pluginError, format(message, payload));
  },
  info: (message: string, payload?: Payload) => {
    devConsole("info", message, payload);
    safe(pluginInfo, format(message, payload));
  },
  warn: (message: string, payload?: Payload) => {
    devConsole("warn", message, payload);
    safe(pluginWarn, format(message, payload));
  },
};
