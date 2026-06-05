import type { FC } from "react";
import type { StorageUsage } from "@/commands";
import { cn } from "@/utils/cn";
import type { PreferenceStorageState } from "../types/preferences";
import {
  formatBytes,
  storageMeterClass,
  storageTargetBytes,
  storageToneClass,
} from "../utils/storageUsage";

interface PreferenceStorageUsagePanelProps {
  state: PreferenceStorageState;
  storageUsage: StorageUsage | null;
}

/**
 * 侧栏里的本地存储摘要，展示当前环境数据目录的递归占用。
 */
const PreferenceStorageUsagePanel: FC<PreferenceStorageUsagePanelProps> = (
  props,
) => {
  const { state, storageUsage } = props;
  const isReady = state === "ready" && storageUsage !== null;
  const totalLabel = storageUsage ? formatBytes(storageUsage.totalBytes) : "--";
  const targetLabel = storageUsage
    ? formatBytes(storageTargetBytes(storageUsage.totalBytes))
    : "--";
  const usageLabel =
    state === "loading" ? "统计中" : `${totalLabel} / ${targetLabel}`;
  const meterClassName = isReady
    ? storageMeterClass(storageUsage.totalBytes)
    : "w-1/5";
  const storageToneClassName = isReady
    ? storageToneClass(storageUsage.totalBytes)
    : { bg: "bg-ant-success", text: "text-ant-success" };

  return (
    <div className="px-3 pb-3">
      <div className="rounded-2 border border-ant-border-secondary bg-ant-fill-quaternary px-3 py-3">
        <div className="flex min-w-0 items-start gap-2.5">
          <span
            className={cn(
              "flex size-7 shrink-0 items-center justify-center text-lg",
              state === "error" ? "text-ant-error" : storageToneClassName.text,
            )}
          >
            <i aria-hidden="true" className="i-lucide:hard-drive" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate font-medium text-ant-text text-sm leading-tight">
              本地存储
            </div>
            <div
              className={cn(
                "mt-1 truncate font-medium text-xs leading-tight",
                state === "error" ? "text-ant-error" : "text-ant-secondary",
              )}
            >
              {state === "error" ? "统计失败" : usageLabel}
            </div>
          </div>
        </div>

        <div className="mt-3 h-1 overflow-hidden rounded-full bg-ant-fill-secondary">
          <span
            className={cn(
              "block h-full rounded-full transition-all motion-reduce:transition-none",
              state === "error" ? "bg-ant-error" : storageToneClassName.bg,
              meterClassName,
            )}
          />
        </div>
      </div>
    </div>
  );
};

export default PreferenceStorageUsagePanel;
