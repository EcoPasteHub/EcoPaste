import { useMount } from "ahooks";
import { Switch } from "antd";
import type { FC } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  checkAccessibilityPermission,
  checkFullDiskAccessPermission,
  requestAccessibilityPermission,
  requestFullDiskAccessPermission,
} from "tauri-plugin-macos-permissions-api";
import { getRunAsAdminStatus, restartAsAdmin, setRunAsAdmin } from "@/commands";
import { getModalApi } from "@/utils/feedback";
import { log } from "@/utils/log";
import type { PreferenceSetting } from "../../types/preferences";
import ControlFrame from "./ControlFrame";

const PERMISSION_POLL_INTERVAL_MS = 1_500;

type PermissionKind = NonNullable<
  Extract<PreferenceSetting["control"], { type: "permission" }>["kind"]
>;
type PermissionStatus = "granted" | "denied" | "unknown" | "notRequired";

interface PermissionState {
  configured: boolean;
  status: PermissionStatus;
}

interface PermissionControlProps {
  disabled: boolean;
  setting: PreferenceSetting;
}

/**
 * 展示系统授权入口，并以系统真实状态刷新开关和状态标签。
 */
const PermissionControl: FC<PermissionControlProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { disabled, setting } = props;
  const [permissionState, setPermissionState] = useState<PermissionState>({
    configured: false,
    status: "unknown",
  });
  const [authorizing, setAuthorizing] = useState(false);
  const checkingRef = useRef(false);
  const isPermissionControl = setting.control.type === "permission";
  const kind = resolvePermissionKind(setting.control);

  const checkPermission = useCallback(async () => {
    if (!isPermissionControl) return;
    if (checkingRef.current) return;

    checkingRef.current = true;

    try {
      setPermissionState(await readPermissionState(kind));
    } catch (error) {
      log.warn("check preference permission failed", error);
    } finally {
      checkingRef.current = false;
    }
  }, [isPermissionControl, kind]);

  const disableAdminLaunch = async () => {
    setAuthorizing(true);

    try {
      await setRunAsAdmin(false);
    } catch (error) {
      log.warn("disable administrator launch failed", error);
    } finally {
      setAuthorizing(false);
      await checkPermission();
    }
  };

  const enableAdminLaunch = () => {
    getModalApi().confirm({
      cancelText: t("common:actions.cancel"),
      centered: true,
      content: t("schema.settings.permissions.adminRestartConfirm.content"),
      okText: t("schema.settings.permissions.adminRestartConfirm.ok"),
      onOk: async () => {
        setAuthorizing(true);

        try {
          await setRunAsAdmin(true);
          await restartAsAdmin();
        } catch (error) {
          log.warn("enable administrator launch failed", error);
        } finally {
          setAuthorizing(false);
          await checkPermission();
        }
      },
      title: t("schema.settings.permissions.adminRestartConfirm.title"),
    });
  };

  const handleChange = async (checked: boolean) => {
    if (kind === "runAsAdministrator") {
      if (checked) {
        enableAdminLaunch();
        return;
      }

      await disableAdminLaunch();
      return;
    }

    if (!checked) return;

    setAuthorizing(true);

    try {
      await requestSystemPermission(kind);
    } catch (error) {
      log.warn("request preference permission failed", error);
    } finally {
      setAuthorizing(false);
      await checkPermission();
    }
  };

  useMount(() => {
    void checkPermission();
  });

  useEffect(() => {
    if (!isPermissionControl) return;
    if (permissionState.status === "granted") return;

    const timer = window.setInterval(() => {
      void checkPermission();
    }, PERMISSION_POLL_INTERVAL_MS);

    return () => {
      window.clearInterval(timer);
    };
  }, [checkPermission, isPermissionControl, permissionState.status]);

  if (!isPermissionControl) return null;

  const checked =
    kind === "runAsAdministrator"
      ? permissionState.configured
      : permissionState.status === "granted";
  const loading = authorizing || permissionState.status === "unknown";

  return (
    <ControlFrame>
      <Switch
        aria-label={t(`schema.settings.${setting.id}.title`)}
        checked={checked}
        disabled={disabled || loading}
        loading={loading}
        onChange={handleChange}
      />
    </ControlFrame>
  );
};

export default PermissionControl;

function resolvePermissionKind(
  control: PreferenceSetting["control"],
): PermissionKind {
  if (control.type !== "permission") return "accessibility";

  return control.kind;
}

async function readPermissionState(
  kind: PermissionKind,
): Promise<PermissionState> {
  if (kind === "runAsAdministrator") {
    const status = await getRunAsAdminStatus();

    return {
      configured: status.configured,
      status: status.runningAsAdmin ? "granted" : "denied",
    };
  }

  if (kind === "accessibility") {
    const granted = await checkAccessibilityPermission();

    return {
      configured: false,
      status: granted ? "granted" : "denied",
    };
  }

  if (kind === "fullDiskAccess") {
    const granted = await checkFullDiskAccessPermission();

    return {
      configured: false,
      status: granted ? "granted" : "denied",
    };
  }

  return {
    configured: false,
    status: "notRequired",
  };
}

async function requestSystemPermission(kind: PermissionKind): Promise<void> {
  if (kind === "accessibility") {
    await requestAccessibilityPermission();
    return;
  }

  if (kind === "fullDiskAccess") {
    await requestFullDiskAccessPermission();
  }
}
