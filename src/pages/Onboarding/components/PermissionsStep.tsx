import { useMount } from "ahooks";
import { Button, Switch } from "antd";
import type { FC } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  checkAccessibilityPermission,
  checkFullDiskAccessPermission,
  requestAccessibilityPermission,
  requestFullDiskAccessPermission,
} from "tauri-plugin-macos-permissions-api";
import { getRunAsAdminStatus, restartAsAdmin, setRunAsAdmin } from "@/commands";
import { getModalApi } from "@/utils/feedback";
import { isMac, isWin } from "@/utils/is";
import { log } from "@/utils/log";
import type { OnboardingStepProps } from "../types";
import OnboardingCard from "./OnboardingCard";
import OnboardingStepLayout from "./OnboardingStepLayout";

const PERMISSION_POLL_INTERVAL_MS = 1_500;

type OnboardingPermissionKind =
  | "accessibility"
  | "fullDiskAccess"
  | "runAsAdministrator";
type OnboardingPermissionStatus =
  | "granted"
  | "denied"
  | "unknown"
  | "notRequired";

interface OnboardingPermissionState {
  configured: boolean;
  kind: OnboardingPermissionKind;
  status: OnboardingPermissionStatus;
}

const PERMISSION_ICONS = {
  accessibility: "i-lucide:accessibility",
  fullDiskAccess: "i-lucide:hard-drive",
  runAsAdministrator: "i-lucide:shield-alert",
} as const;

const PermissionsStep: FC<OnboardingStepProps> = () => {
  const { t } = useTranslation(["onboarding", "common"]);
  const [permissions, setPermissions] = useState<OnboardingPermissionState[]>(
    () => {
      return buildInitialPermissions();
    },
  );
  const [authorizingKind, setAuthorizingKind] =
    useState<OnboardingPermissionKind | null>(null);
  const checkingRef = useRef(false);

  const allGranted = useMemo(() => {
    return permissions.every((permission) => {
      return (
        permission.status === "granted" || permission.status === "notRequired"
      );
    });
  }, [permissions]);

  const checkPermissions = useCallback(async () => {
    if (checkingRef.current) return;

    checkingRef.current = true;

    try {
      setPermissions(await readPermissionStates());
    } catch (error) {
      log.warn("check onboarding permissions failed", error);
    } finally {
      checkingRef.current = false;
    }
  }, []);

  const handleAuthorize = useCallback(
    async (kind: OnboardingPermissionKind) => {
      setAuthorizingKind(kind);

      try {
        await requestSystemPermission(kind);
      } catch (error) {
        log.warn("request onboarding permission failed", error);
      } finally {
        setAuthorizingKind(null);
        await checkPermissions();
      }
    },
    [checkPermissions],
  );

  const handleAdminLaunchChange = useCallback(
    async (enabled: boolean) => {
      if (!enabled) {
        setAuthorizingKind("runAsAdministrator");

        try {
          await setRunAsAdmin(false);
        } catch (error) {
          log.warn("disable administrator launch failed", error);
        } finally {
          setAuthorizingKind(null);
          await checkPermissions();
        }

        return;
      }

      getModalApi().confirm({
        cancelText: t("common:actions.cancel"),
        centered: true,
        content: t("permissions.adminRestartConfirm.content"),
        okText: t("permissions.adminRestartConfirm.ok"),
        onOk: async () => {
          setAuthorizingKind("runAsAdministrator");

          try {
            await setRunAsAdmin(true);
            await restartAsAdmin();
          } catch (error) {
            log.warn("enable administrator launch failed", error);
          } finally {
            setAuthorizingKind(null);
            await checkPermissions();
          }
        },
        title: t("permissions.adminRestartConfirm.title"),
      });
    },
    [checkPermissions, t],
  );

  useMount(() => {
    void checkPermissions();
  });

  useEffect(() => {
    if (allGranted) return;

    const timer = window.setInterval(() => {
      void checkPermissions();
    }, PERMISSION_POLL_INTERVAL_MS);

    return () => {
      window.clearInterval(timer);
    };
  }, [allGranted, checkPermissions]);

  return (
    <OnboardingStepLayout
      contentClassName="flex flex-col gap-4"
      description={t(`permissions.description.${isMac ? "macos" : "windows"}`)}
      icon={<i aria-hidden="true" className="i-lucide:shield-check" />}
      title={t("permissions.title")}
    >
      {permissions.map((permission) => {
        return (
          <PermissionCard
            authorizing={authorizingKind === permission.kind}
            key={permission.kind}
            onAdminLaunchChange={handleAdminLaunchChange}
            onAuthorize={handleAuthorize}
            permission={permission}
          />
        );
      })}
    </OnboardingStepLayout>
  );
};

export default PermissionsStep;

interface PermissionCardProps {
  authorizing: boolean;
  onAdminLaunchChange: (enabled: boolean) => void;
  onAuthorize: (kind: OnboardingPermissionKind) => void;
  permission: OnboardingPermissionState;
}

const PermissionCard: FC<PermissionCardProps> = (props) => {
  const { authorizing, onAdminLaunchChange, onAuthorize, permission } = props;
  const { t } = useTranslation("onboarding");

  const handleAuthorizeClick = () => {
    onAuthorize(permission.kind);
  };

  const handleAdminSwitchChange = (checked: boolean) => {
    onAdminLaunchChange(checked);
  };

  return (
    <OnboardingCard
      action={
        permission.kind === "runAsAdministrator" ? (
          <Switch
            aria-label={t(`permissions.items.${permission.kind}.title`)}
            checked={permission.configured}
            disabled={permission.status === "unknown" || authorizing}
            loading={authorizing}
            onChange={handleAdminSwitchChange}
          />
        ) : (
          <Button
            color={getPermissionButtonColor(permission.status)}
            loading={authorizing}
            onClick={
              permission.status === "denied" ? handleAuthorizeClick : void 0
            }
            tabIndex={permission.status === "denied" ? void 0 : -1}
            variant="outlined"
          >
            {permission.status === "denied"
              ? t("permissions.action.authorize")
              : t(`permissions.status.${permission.status}`)}
          </Button>
        )
      }
      description={t(`permissions.items.${permission.kind}.description`)}
      icon={PERMISSION_ICONS[permission.kind]}
      title={t(`permissions.items.${permission.kind}.title`)}
    />
  );
};

function getPermissionButtonColor(status: OnboardingPermissionStatus) {
  if (status === "granted" || status === "notRequired") {
    return "green";
  }

  if (status === "denied") {
    return "danger";
  }

  if (status === "unknown") {
    return "primary";
  }

  return "default";
}

function buildInitialPermissions(): OnboardingPermissionState[] {
  if (isWin) {
    return [
      {
        configured: false,
        kind: "runAsAdministrator",
        status: "unknown",
      },
    ];
  }

  if (!isMac) {
    return [
      {
        configured: false,
        kind: "runAsAdministrator",
        status: "notRequired",
      },
    ];
  }

  return [
    {
      configured: false,
      kind: "accessibility",
      status: "unknown",
    },
    {
      configured: false,
      kind: "fullDiskAccess",
      status: "unknown",
    },
  ];
}

async function readPermissionStates(): Promise<OnboardingPermissionState[]> {
  if (isWin) {
    const status = await getRunAsAdminStatus();

    return [
      {
        configured: status.configured,
        kind: "runAsAdministrator",
        status: status.runningAsAdmin ? "granted" : "denied",
      },
    ];
  }

  if (!isMac) {
    return [
      {
        configured: false,
        kind: "runAsAdministrator",
        status: "notRequired",
      },
    ];
  }

  const [accessibilityGranted, fullDiskAccessGranted] = await Promise.all([
    checkAccessibilityPermission(),
    checkFullDiskAccessPermission(),
  ]);

  return [
    {
      configured: false,
      kind: "accessibility",
      status: accessibilityGranted ? "granted" : "denied",
    },
    {
      configured: false,
      kind: "fullDiskAccess",
      status: fullDiskAccessGranted ? "granted" : "denied",
    },
  ];
}

async function requestSystemPermission(
  kind: OnboardingPermissionKind,
): Promise<void> {
  if (kind === "accessibility") {
    await requestAccessibilityPermission();
    return;
  }

  if (kind === "fullDiskAccess") {
    await requestFullDiskAccessPermission();
  }
}
