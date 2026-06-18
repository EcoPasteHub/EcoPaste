import { useMount } from "ahooks";
import { Tag } from "antd";
import type { FC } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  checkAccessibilityPermission,
  checkFullDiskAccessPermission,
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

const STATUS_ICONS = {
  denied: "i-lucide:circle-alert text-base",
  granted: "i-lucide:circle-check text-base",
  notRequired: "i-lucide:circle-check text-base",
  pendingIntegration: "i-lucide:circle-dashed text-base",
  unknown: "i-lucide:circle-help text-base",
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
        return <PermissionCard key={permission.kind} permission={permission} />;
      })}
    </OnboardingStepLayout>
  );
};

export default PermissionsStep;

interface PermissionCardProps {
  permission: OnboardingPermissionState;
}

const PermissionCard: FC<PermissionCardProps> = (props) => {
  const { permission } = props;
  const { t } = useTranslation("onboarding");

  return (
    <OnboardingCard
      action={
        <Tag
          className="m-0"
          color={getPermissionStatusColor(permission.status)}
          icon={
            <i aria-hidden="true" className={STATUS_ICONS[permission.status]} />
          }
          variant="outlined"
        >
          {t(`permissions.status.${permission.status}`)}
        </Tag>
      }
      description={t(`permissions.items.${permission.kind}.description`)}
      icon={PERMISSION_ICONS[permission.kind]}
      title={t(`permissions.items.${permission.kind}.title`)}
    />
  );
};

function getPermissionStatusColor(status: OnboardingPermissionStatus) {
  if (status === "granted" || status === "notRequired") {
    return "success";
  }

  if (status === "denied") {
    return "error";
  }

  if (status === "unknown") {
    return "processing";
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
