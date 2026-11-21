import { useMount } from "ahooks";
import { Select } from "antd";
import type { FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { listClipboardGroups } from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import {
  parseWindowOpenGroupId,
  toWindowOpenGroupValue,
  WINDOW_OPEN_SELECTION_ALL,
  WINDOW_OPEN_SELECTION_PRESERVE,
} from "@/constants/windowOpenSelection";
import { useTauriListen } from "@/hooks/useTauriListen";
import type { ClipboardGroupRecord } from "@/types/clipboard";
import type { PreferenceSetting } from "../../types/preferences";
import type { ControlProps } from "./types";

interface ClipboardGroupSelectControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: string;
}

/**
 * Selects the custom group that the clipboard window should activate on open.
 */
const ClipboardGroupSelectControl: FC<ClipboardGroupSelectControlProps> = (
  props,
) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, value } = props;
  const [groups, setGroups] = useState<ClipboardGroupRecord[]>([]);
  const [loading, setLoading] = useState(false);

  const loadGroups = async () => {
    setLoading(true);

    try {
      const records = await listClipboardGroups();

      setGroups(records);
    } catch {
      // `listClipboardGroups` already logs and shows the localized command toast.
    } finally {
      setLoading(false);
    }
  };

  const handleGroupsUpdated = () => {
    void loadGroups();
  };

  useMount(() => {
    void loadGroups();
  });

  useTauriListen(TAURI_EVENT.CLIPBOARD_GROUPS_UPDATED, handleGroupsUpdated);

  const options = [
    {
      label: t("schema.settings.window.selectGroupOnOpen.options.preserve"),
      value: WINDOW_OPEN_SELECTION_PRESERVE,
    },
    {
      label: t("schema.settings.window.selectGroupOnOpen.options.all"),
      value: WINDOW_OPEN_SELECTION_ALL,
    },
    ...groups.map((group) => {
      return {
        label: group.name,
        value: toWindowOpenGroupValue(group.id),
      };
    }),
  ];

  const selectedGroupId = parseWindowOpenGroupId(value);
  const selectedGroupExists = groups.some((group) => {
    return group.id === selectedGroupId;
  });

  if (selectedGroupId && !selectedGroupExists) {
    options.push({
      label: t("schema.settings.window.selectGroupOnOpen.options.missingGroup"),
      value,
    });
  }

  const handleChange = async (next: string) => {
    await onChange(setting, next);
  };

  return (
    <Select
      disabled={disabled}
      loading={loading}
      onChange={handleChange}
      options={options}
      value={value}
    />
  );
};

export default ClipboardGroupSelectControl;
