import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProSelect from "@/components/ProSelect";
import { LANGUAGE } from "@/constants";
import { useImmediateKey } from "@/hooks/useImmediateKey";
import { globalStore } from "@/stores/global";
import { raf } from "@/utils/bom";

interface Option {
  label: string;
  value: string;
}

const Language = () => {
  const { appearance } = useSnapshot(globalStore);
  const { t } = useTranslation();

  useImmediateKey(globalStore.appearance, "language", () => {
    const appWindow = getCurrentWebviewWindow();

    raf(() => {
      appWindow.setTitle(t("preference.title"));
    });
  });

  const options: Option[] = [
    {
      label: "简体中文",
      value: LANGUAGE.ZH_CN,
    },
    {
      label: "繁體中文",
      value: LANGUAGE.ZH_TW,
    },
    {
      label: "English",
      value: LANGUAGE.EN_US,
    },
    {
      label: "日本語",
      value: LANGUAGE.JA_JP,
    },
  ];

  return (
    <ProSelect
      onChange={(value) => {
        globalStore.appearance.language = value;
      }}
      options={options}
      title={t("preference.settings.appearance_settings.label.language")}
      value={appearance.language}
    />
  );
};

export default Language;
