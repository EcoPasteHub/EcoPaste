import { useMount } from "ahooks";
import { Alert, Button, Input, Modal, Switch, Tag } from "antd";
import type { ChangeEvent, FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import {
  type CloudProbeResult,
  type CloudStatus,
  cloudGetStatus,
  cloudLogin,
  cloudLogout,
  cloudProbe,
  cloudPullRecent,
  updateSettings,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import { settingsState } from "@/stores/settings";
import type { Settings } from "@/types/settings";
import { getMessageApi } from "@/utils/feedback";
import { log } from "@/utils/log";

type StateColor = "default" | "error" | "processing" | "success";

const STATE_COLOR: Record<CloudStatus["state"], StateColor> = {
  connected: "success",
  connecting: "processing",
  disconnected: "default",
  error: "error",
};

/**
 * 「云同步」分组：连接自托管服务器（证书指纹确认 → 登录），开关、状态与拉取历史。
 */
const CloudSyncPanel: FC = () => {
  const { t } = useTranslation("preferences");
  const settings = useSnapshot(settingsState) as Settings;
  const [status, setStatus] = useState<CloudStatus | null>(null);
  const [serverUrl, setServerUrl] = useState("");
  const [probe, setProbe] = useState<CloudProbeResult | null>(null);
  const [loginOpen, setLoginOpen] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);

  const enabled = settings.cloud?.enabled === true;
  const loggedIn = status?.loggedIn === true;

  const refresh = async () => {
    try {
      setStatus(await cloudGetStatus());
    } catch (error) {
      log.warn("load cloud status failed", error);
    }
  };

  const handleToggle = async (checked: boolean) => {
    try {
      await updateSettings({ cloud: { enabled: checked } });
    } catch (error) {
      log.warn("toggle cloud sync failed", error);
    }
  };

  const handleServerChange = (event: ChangeEvent<HTMLInputElement>) => {
    setServerUrl(event.target.value);
  };

  const handleConnect = async () => {
    if (serverUrl.trim().length === 0) return;

    setBusy(true);
    try {
      setProbe(await cloudProbe(serverUrl));
    } catch (error) {
      log.warn("probe cloud server failed", error);
    } finally {
      setBusy(false);
    }
  };

  const handleTrust = () => {
    setLoginOpen(true);
  };

  const handleCancelTrust = () => {
    setProbe(null);
  };

  const handleCancelLogin = () => {
    if (busy) return;

    setLoginOpen(false);
    setPassword("");
  };

  const handleUsernameChange = (event: ChangeEvent<HTMLInputElement>) => {
    setUsername(event.target.value);
  };

  const handlePasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    setPassword(event.target.value);
  };

  const handleLogin = async () => {
    if (!probe || username.trim().length === 0 || password.length === 0) return;

    setBusy(true);
    try {
      const next = await cloudLogin({
        fingerprint: probe.fingerprint,
        password,
        serverUrl: probe.serverUrl,
        username,
      });

      setStatus(next);
      setLoginOpen(false);
      setProbe(null);
      setPassword("");
      if (!enabled) {
        await updateSettings({ cloud: { enabled: true } });
      }
    } catch (error) {
      log.warn("cloud login failed", error);
      setPassword("");
    } finally {
      setBusy(false);
    }
  };

  const handleLogout = async () => {
    try {
      await cloudLogout();
      await refresh();
    } catch (error) {
      log.warn("cloud logout failed", error);
    }
  };

  const handlePull = async () => {
    setBusy(true);
    try {
      const count = await cloudPullRecent();

      getMessageApi().success(t("sync.cloud.pulled", { count }));
    } catch (error) {
      log.warn("pull cloud history failed", error);
    } finally {
      setBusy(false);
    }
  };

  useMount(async () => {
    await refresh();
  });

  useTauriListen(TAURI_EVENT.CLOUD_STATUS, () => {
    void refresh();
  });

  useTauriListen(TAURI_EVENT.SETTINGS_UPDATED, () => {
    void refresh();
  });

  return (
    <div className="flex flex-col gap-3 px-4 py-3">
      <span className="text-ant-secondary text-xs">
        {t("sync.cloud.description")}
      </span>

      <div className="flex items-center justify-between gap-3">
        <span className="text-ant-text text-sm">{t("sync.cloud.enable")}</span>
        <Switch
          checked={enabled}
          disabled={!loggedIn}
          onChange={handleToggle}
        />
      </div>

      {loggedIn && status ? (
        <div className="flex flex-col gap-2 rounded-2 border border-ant-border-secondary bg-ant-layout p-3">
          <div className="flex items-center justify-between gap-2">
            <span className="text-ant-text text-sm">
              {t("sync.cloud.loggedInAs", { username: status.username })}
            </span>
            <Tag color={STATE_COLOR[status.state]} variant="filled">
              {t(`sync.cloud.state.${status.state}`)}
            </Tag>
          </div>
          <span className="break-all text-ant-secondary text-xs">
            {t("sync.cloud.server", { server: status.serverUrl })}
          </span>
          <span className="break-all text-ant-tertiary text-xs">
            {t("sync.cloud.fingerprint")}: {status.fingerprint}
          </span>
          {status.state === "error" && status.error && (
            <span className="text-ant-error text-xs">{status.error}</span>
          )}
          <div className="flex gap-2">
            <Button
              disabled={busy}
              onClick={handlePull}
              size="small"
              type="primary"
            >
              {t("sync.cloud.pull")}
            </Button>
            <Button danger onClick={handleLogout} size="small">
              {t("sync.cloud.logout")}
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          <span className="text-ant-secondary text-xs">
            {t("sync.cloud.serverLabel")}
          </span>
          <div className="flex gap-2">
            <Input
              disabled={busy}
              onChange={handleServerChange}
              onPressEnter={handleConnect}
              placeholder={t("sync.cloud.serverPlaceholder")}
              value={serverUrl}
            />
            <Button
              disabled={serverUrl.trim().length === 0}
              loading={busy && !loginOpen}
              onClick={handleConnect}
              type="primary"
            >
              {t("sync.cloud.connect")}
            </Button>
          </div>
        </div>
      )}

      <Alert showIcon title={t("sync.cloud.warning")} type="warning" />

      <Modal
        cancelText={t("sync.devices.cancel")}
        centered
        okText={t("sync.cloud.trustAction")}
        onCancel={handleCancelTrust}
        onOk={handleTrust}
        open={probe !== null && !loginOpen}
        title={t("sync.cloud.trustTitle")}
        width={460}
      >
        <div className="flex flex-col gap-3 py-2">
          <span className="text-ant-secondary text-sm leading-relaxed">
            {t("sync.cloud.trustHint")}
          </span>
          <div className="break-all rounded-2 border border-ant-border bg-ant-layout p-3 font-mono text-ant-text text-xs leading-relaxed">
            {probe?.fingerprint}
          </div>
          <span className="break-all text-ant-secondary text-xs">
            {t("sync.cloud.server", { server: probe?.serverUrl ?? "" })}
          </span>
        </div>
      </Modal>

      <Modal
        cancelText={t("sync.devices.cancel")}
        centered
        confirmLoading={busy}
        okButtonProps={{
          disabled: username.trim().length === 0 || password.length === 0,
        }}
        okText={t("sync.cloud.login")}
        onCancel={handleCancelLogin}
        onOk={handleLogin}
        open={loginOpen}
        title={t("sync.cloud.loginTitle")}
        width={400}
      >
        <div className="flex flex-col gap-3 py-2">
          <Input
            autoComplete="username"
            onChange={handleUsernameChange}
            placeholder={t("sync.cloud.username")}
            value={username}
          />
          <Input.Password
            autoComplete="current-password"
            onChange={handlePasswordChange}
            onPressEnter={handleLogin}
            placeholder={t("sync.cloud.password")}
            value={password}
          />
        </div>
      </Modal>
    </div>
  );
};

export default CloudSyncPanel;
