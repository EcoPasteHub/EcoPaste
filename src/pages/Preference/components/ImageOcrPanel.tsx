import { useRequest, useUnmount } from "ahooks";
import { Alert, Button, Modal, Progress, Spin, Switch, Tag } from "antd";
import type { FC, ReactNode } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  clearImageOcrIndex,
  getImageOcrStatus,
  type ImageOcrStatus,
  queueImageOcrHistory,
} from "@/commands";
import { updateSettings } from "@/stores/settings";
import type { Settings } from "@/types/settings";
import { cn } from "@/utils/cn";

interface ImageOcrPanelProps {
  highlightedSettingId: string | null;
  settings: Settings;
}

/** Local image recognition controls; job scheduling and recognition stay in Rust. */
const ImageOcrPanel: FC<ImageOcrPanelProps> = (props) => {
  const { highlightedSettingId, settings } = props;
  const { t } = useTranslation(["preferences", "common"]);
  const { enabled, paused } = settings.clipboard.ocr;
  const [busy, setBusy] = useState(false);
  const [clearOpen, setClearOpen] = useState(false);
  const { data, error, loading, cancel, mutate, refresh } = useRequest(
    getImageOcrStatus,
    {
      pollingErrorRetryCount: 0,
      pollingInterval: 2000,
      pollingWhenHidden: false,
      refreshDeps: [enabled, paused],
    },
  );

  useUnmount(cancel);

  const unsupported = data?.supported === false;
  const disabled = unsupported || !data || Boolean(error) || busy;
  const processed = (data?.completed ?? 0) + (data?.failed ?? 0);
  const percent = data?.total ? Math.round((processed / data.total) * 100) : 0;

  /** Cancel stale polling results before mutating, then resume from a fresh snapshot. */
  const performAction = async (action: () => Promise<ImageOcrStatus>) => {
    cancel();
    setBusy(true);

    try {
      mutate(await action());
      return true;
    } catch {
      // Command wrappers already show the localized error and log its cause.
      return false;
    } finally {
      setBusy(false);
      refresh();
    }
  };

  const handleEnabledChange = async (checked: boolean) => {
    await performAction(async () => {
      await updateSettings({ clipboard: { ocr: { enabled: checked } } });
      return await getImageOcrStatus();
    });
  };

  const togglePaused = async () => {
    await performAction(async () => {
      await updateSettings({ clipboard: { ocr: { paused: !paused } } });
      return await getImageOcrStatus();
    });
  };

  const queueHistory = async () => {
    await performAction(queueImageOcrHistory);
  };

  const openClear = () => {
    setClearOpen(true);
  };

  const closeClear = () => {
    if (!busy) setClearOpen(false);
  };

  const clearIndex = async () => {
    if (await performAction(clearImageOcrIndex)) setClearOpen(false);
  };

  const refreshStatus = () => {
    refresh();
  };

  const renderRow = (key: string, control: ReactNode) => {
    const id = `imageOcr.${key}`;

    return (
      <div
        className={cn("flex items-center gap-4 border-ant-split border-b p-4", {
          "bg-ant-primary-bg": highlightedSettingId === id,
        })}
        data-preference-setting-id={id}
      >
        <div className="min-w-0 flex-1">
          <div className="font-medium text-sm">
            {t(`schema.settings.${id}.title`)}
          </div>
          <div className="mt-1 text-ant-secondary text-xs leading-relaxed">
            {t(`schema.settings.${id}.description`)}
          </div>
        </div>
        <div className="shrink-0">{control}</div>
      </div>
    );
  };

  return (
    <div>
      <div className="space-y-3 p-4">
        <p className="m-0 text-ant-secondary text-sm leading-relaxed">
          {t("imageOcr.localHint")}
        </p>
        {unsupported ? (
          <Alert showIcon title={t("imageOcr.unsupported")} type="info" />
        ) : null}
        {error ? (
          <Alert
            action={
              <Button onClick={refreshStatus}>
                {t("imageOcr.retryStatus")}
              </Button>
            }
            showIcon
            title={t("imageOcr.statusError")}
            type="error"
          />
        ) : null}
        {!data && loading ? <Spin size="small" /> : null}
        {data ? (
          <div aria-live="polite">
            <Tag>
              {t(
                unsupported
                  ? "imageOcr.unavailable"
                  : !enabled
                    ? "imageOcr.disabled"
                    : paused
                      ? "imageOcr.paused"
                      : "imageOcr.active",
              )}
            </Tag>
            <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-ant-secondary text-xs">
              <span>{t("imageOcr.total", { count: data.total })}</span>
              <span>{t("imageOcr.pending", { count: data.pending })}</span>
              <span>{t("imageOcr.completed", { count: data.completed })}</span>
              <span>{t("imageOcr.failed", { count: data.failed })}</span>
            </div>
            <Progress
              percent={percent}
              showInfo={false}
              size="small"
              status="normal"
            />
            <span className="text-ant-secondary text-xs">
              {t("imageOcr.progress", { processed, total: data.total })}
            </span>
          </div>
        ) : null}
      </div>
      {renderRow(
        "enabled",
        <Switch
          aria-label={t("schema.settings.imageOcr.enabled.title")}
          checked={enabled}
          disabled={disabled}
          loading={busy}
          onChange={handleEnabledChange}
        />,
      )}
      {renderRow(
        "paused",
        <Button disabled={disabled || !enabled} onClick={togglePaused}>
          {t(paused ? "imageOcr.resume" : "imageOcr.pause")}
        </Button>,
      )}
      {renderRow(
        "history",
        <Button disabled={disabled || !enabled} onClick={queueHistory}>
          {t("schema.settings.imageOcr.history.controlLabel")}
        </Button>,
      )}
      {renderRow(
        "clear",
        <Button
          danger
          disabled={disabled || data?.total === 0}
          onClick={openClear}
        >
          {t("schema.settings.imageOcr.clear.controlLabel")}
        </Button>,
      )}
      <Modal
        cancelButtonProps={{ disabled: busy }}
        cancelText={t("common:actions.cancel")}
        closable={!busy}
        confirmLoading={busy}
        okButtonProps={{ danger: true, disabled }}
        okText={t("schema.settings.imageOcr.clear.controlLabel")}
        onCancel={closeClear}
        onOk={clearIndex}
        open={clearOpen}
        title={t("imageOcr.clearTitle")}
      >
        {t("imageOcr.clearDescription")}
      </Modal>
    </div>
  );
};

export default ImageOcrPanel;
