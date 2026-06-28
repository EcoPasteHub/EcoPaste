import { proxy } from "valtio";
import { listAllApps } from "@/commands";
import type { ClipboardApp } from "@/types/clipboard";
import { log } from "@/utils/log";

interface SourceAppsState {
  apps: ClipboardApp[];
  loading: boolean;
}

export const sourceAppsState = proxy<SourceAppsState>({
  apps: [],
  loading: false,
});

let sourceAppsPreload: Promise<void> | null = null;
let sourceAppsLoaded = false;
let sourceAppsRequestToken = 0;

/**
 * 预热应用过滤数据，供偏好窗口后台创建完成后先行填充列表。
 */
export function preloadSourceApps() {
  if (sourceAppsLoaded) return Promise.resolve();
  if (sourceAppsPreload) return sourceAppsPreload;

  sourceAppsPreload = runSourceAppsPreload();

  return sourceAppsPreload;
}

/**
 * 后台刷新完整可过滤应用，并保留右侧已忽略应用的真实元数据。
 */
export async function refreshSourceApps(preservedIds: string[]) {
  await replaceSourceApps(listAllApps(), "refresh source apps failed", {
    preservedIds,
  });
}

/**
 * 重新拉取完整可过滤应用列表，包含当前运行中与已落库来源应用。
 */
export async function reloadSourceApps() {
  await loadSourceApps();
}

/**
 * 合并单个新应用，保持手动添加后左右列表立即可见。
 */
export function mergeSourceApp(app: ClipboardApp) {
  const merged = new Map(
    sourceAppsState.apps.map((item) => {
      return [item.id, item];
    }),
  );
  merged.set(app.id, app);
  sourceAppsState.apps = Array.from(merged.values());
  sourceAppsLoaded = true;
}

/**
 * 从偏好页全局镜像移除已删除的来源应用。
 */
export function removeSourceApps(ids: string[]) {
  if (ids.length === 0) return;

  const removed = new Set(ids);
  sourceAppsState.apps = sourceAppsState.apps.filter((app) => {
    return !removed.has(app.id);
  });
}

/**
 * 拉取全部已知来源应用并写入偏好页全局镜像。
 */
async function loadSourceApps() {
  await replaceSourceApps(listAllApps(), "load source apps failed");
}

/**
 * 执行一次共享预热请求，完成后释放复用中的 Promise。
 */
async function runSourceAppsPreload() {
  try {
    await loadSourceApps();
  } finally {
    sourceAppsPreload = null;
  }
}

/**
 * 写入最新一次来源应用请求的结果，避免较慢的旧请求覆盖新刷新。
 */
async function replaceSourceApps(
  request: Promise<ClipboardApp[]>,
  errorMessage: string,
  options: { preservedIds?: string[] } = {},
) {
  const token = sourceAppsRequestToken + 1;
  sourceAppsRequestToken = token;
  sourceAppsState.loading = true;

  try {
    const apps = await request;
    if (sourceAppsRequestToken !== token) return;

    sourceAppsState.apps = mergePreservedApps(apps, options.preservedIds ?? []);
    sourceAppsLoaded = true;
  } catch (error) {
    if (sourceAppsRequestToken !== token) return;

    log.warn(errorMessage, error);
  } finally {
    if (sourceAppsRequestToken === token) {
      sourceAppsState.loading = false;
    }
  }
}

/**
 * 用最新可过滤应用替换左侧列表，同时保留已忽略应用已有的完整行数据。
 */
function mergePreservedApps(apps: ClipboardApp[], preservedIds: string[]) {
  if (preservedIds.length === 0) return apps;

  const merged = new Map(
    apps.map((app) => {
      return [app.id, app];
    }),
  );
  const preserved = new Set(preservedIds);

  for (const app of sourceAppsState.apps) {
    if (!preserved.has(app.id)) continue;
    if (merged.has(app.id)) continue;

    merged.set(app.id, app);
  }

  return Array.from(merged.values());
}
