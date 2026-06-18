import { useMount } from "ahooks";
import { Button } from "antd";
import type { FC } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  checkAccessibilityPermission,
  checkFullDiskAccessPermission,
  requestAccessibilityPermission,
  requestFullDiskAccessPermission,
} from "tauri-plugin-macos-permissions-api";
import { isMac } from "@/utils/is";
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
  | "notRequired"
  | "pendingIntegration";

interface OnboardingPermissionState {
  kind: OnboardingPermissionKind;
  status: OnboardingPermissionStatus;
}

const PERMISSION_ICONS = {
  accessibility: "i-lucide:accessibility",
  fullDiskAccess: "i-lucide:hard-drive",
  runAsAdministrator: "i-lucide:shield-alert",
} as const;

const PermissionsStep: FC<OnboardingStepProps> = () => {
  const { t } = useTranslation("onboarding");
  const [permissions, setPermissions] = useState<OnboardingPermissionState[]>(
    () => {
      return buildInitialPermissions();
    },
  );
  const checkingRef = useRef(false);

  const allGranted = useMemo(() => {
    return permissions.every((permission) => {
      return (
        permission.status === "granted" ||
        permission.status === "notRequired" ||
        permission.status === "pendingIntegration"
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
      try {
        await requestPermission(kind);
        await checkPermissions();
      } catch (error) {
        log.warn("request onboarding permission failed", error);
      }
    },
    [checkPermissions],
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
            key={permission.kind}
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
  onAuthorize: (kind: OnboardingPermissionKind) => void;
  permission: OnboardingPermissionState;
}

const PermissionCard: FC<PermissionCardProps> = (props) => {
  const { onAuthorize, permission } = props;
  const { t } = useTranslation("onboarding");

  const handleAuthorizeClick = () => {
    onAuthorize(permission.kind);
  };

  return (
    <OnboardingCard
      action={
        <Button
          color={getPermissionButtonColor(permission.status)}
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
  if (!isMac) {
    return [
      {
        kind: "runAsAdministrator",
        status: "pendingIntegration",
      },
    ];
  }

  return [
    {
      kind: "accessibility",
      status: "unknown",
    },
    {
      kind: "fullDiskAccess",
      status: "unknown",
    },
  ];
}

async function readPermissionStates(): Promise<OnboardingPermissionState[]> {
  if (!isMac) {
    return [
      {
        kind: "runAsAdministrator",
        status: "pendingIntegration",
      },
    ];
  }

  const [accessibilityGranted, fullDiskAccessGranted] = await Promise.all([
    checkAccessibilityPermission(),
    checkFullDiskAccessPermission(),
  ]);

  return [
    {
      kind: "accessibility",
      status: accessibilityGranted ? "granted" : "denied",
    },
    {
      kind: "fullDiskAccess",
      status: fullDiskAccessGranted ? "granted" : "denied",
    },
  ];
}

async function requestPermission(
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
