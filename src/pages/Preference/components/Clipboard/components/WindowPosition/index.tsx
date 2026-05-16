import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProSelect from "@/components/ProSelect";
import { clipboardStore } from "@/stores/clipboard";
import type { ClipboardStore } from "@/types/store";

interface Option {
  label: string;
  value: ClipboardStore["window"]["position"];
}

const WindowPosition = () => {
  const { window } = useSnapshot(clipboardStore);
  const { t } = useTranslation();

  const options: Option[] = [
    {
      label: t(
        "preference.clipboard.window_settings.label.window_position_remember",
      ),
      value: "remember",
    },
    {
      label: t(
        "preference.clipboard.window_settings.label.window_position_follow",
      ),
      value: "follow",
    },
    {
      label: t(
        "preference.clipboard.window_settings.label.window_position_center",
      ),
      value: "center",
    },
  ];

  return (
    <ProSelect
      onChange={(value) => {
        clipboardStore.window.position = value;
      }}
      options={options}
      title={t("preference.clipboard.window_settings.label.window_position")}
      value={window.position}
    />
  );
};

export default WindowPosition;
