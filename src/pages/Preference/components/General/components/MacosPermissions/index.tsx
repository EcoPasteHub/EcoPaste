import { confirm } from "@tauri-apps/plugin-dialog";
import { useMount, useReactive } from "ahooks";
import { useTranslation } from "react-i18next";
import {
  checkAccessibilityPermission,
  checkFullDiskAccessPermission,
  requestAccessibilityPermission,
  requestFullDiskAccessPermission,
} from "tauri-plugin-macos-permissions-api";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import UnoIcon from "@/components/UnoIcon";

const MacosPermissions = () => {
  const { t } = useTranslation();

  const state = useReactive({
    accessibilityPermission: false,
    fullDiskAccessPermission: false,
  });

  useMount(() => {
    checkAccessibility();

    checkFullDiskAccess();
  });

  const checkAccessibility = async () => {
    await requestAccessibilityPermission();

    const check = async () => {
      state.accessibilityPermission = await checkAccessibilityPermission();

      if (state.accessibilityPermission) return;

      setTimeout(check, 1000);
    };

    check();
  };

  const checkFullDiskAccess = async () => {
    state.fullDiskAccessPermission = await checkFullDiskAccessPermission();

    if (state.fullDiskAccessPermission) return;

    const confirmed = await confirm(
      t(
        "preference.settings.permission_settings.hints.confirm_full_disk_access",
      ),
      {
        cancelLabel: t(
          "preference.settings.permission_settings.button.cancel_full_disk_access",
        ),
        okLabel: t(
          "preference.settings.permission_settings.button.confirm_full_disk_access",
        ),
        title: t(
          "preference.settings.permission_settings.label.confirm_full_disk_access",
        ),
      },
    );

    if (!confirmed) return;

    requestFullDiskAccessPermission();
  };

  const renderStatus = (authorized: boolean, event: () => Promise<void>) => {
    return (
      <div className="children:(inline-flex items-center gap-1 font-bold)">
        {authorized ? (
          <div className="text-primary">
            <UnoIcon name="i-lucide:circle-check" />
            {t("preference.settings.permission_settings.label.authorized")}
          </div>
        ) : (
          <div className="cursor-pointer text-danger" onMouseDown={event}>
            <UnoIcon name="i-lucide:circle-arrow-right" />
            {t("preference.settings.permission_settings.button.authorize")}
          </div>
        )}
      </div>
    );
  };

  return (
    <ProList header={t("preference.settings.permission_settings.title")}>
      <ProListItem
        description={t(
          "preference.settings.permission_settings.hints.accessibility_permissions",
        )}
        title={t(
          "preference.settings.permission_settings.label.accessibility_permissions",
        )}
      >
        {renderStatus(state.accessibilityPermission, checkAccessibility)}
      </ProListItem>

      <ProListItem
        description={t(
          "preference.settings.permission_settings.hints.full_disk_access_permissions",
        )}
        title={t(
          "preference.settings.permission_settings.label.full_disk_access_permissions",
        )}
      >
        {renderStatus(state.fullDiskAccessPermission, checkFullDiskAccess)}
      </ProListItem>
    </ProList>
  );
};

export default MacosPermissions;
