import { useSnapshot } from "valtio";
import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";

interface Option {
  label: string;
  value: ClipboardStore["content"]["autoPaste"];
}

const AutoPaste = () => {
  const { content } = useSnapshot(clipboardStore);
  const { t } = useTranslation();

  const options: Option[] = [
    {
      label: t("preference.clipboard.content_settings.label.auto_paste_single"),
      value: "single",
    },
    {
      label: t("preference.clipboard.content_settings.label.auto_paste_double"),
      value: "double",
    },
  ];

  return (
    <ProSelect
      description={t("preference.clipboard.content_settings.hints.auto_paste")}
      onChange={(value) => {
        clipboardStore.content.autoPaste = value;
      }}
      options={options}
      title={t("preference.clipboard.content_settings.label.auto_paste")}
      value={content.autoPaste}
    />
  );
};

export default AutoPaste;
