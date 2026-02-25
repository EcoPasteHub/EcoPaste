import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProSelect from "@/components/ProSelect";
import { clipboardStore } from "@/stores/clipboard";

interface Option {
  label: string;
  value: number;
}

const DisplayLines = () => {
  const { content } = useSnapshot(clipboardStore);
  const { t } = useTranslation();

  const options: Option[] = Array.from({ length: 20 }, (_, i) => i + 1).map(
    (value) => ({
      label: String(value),
      value,
    }),
  );

  return (
    <ProSelect
      className="w-30"
      description={t(
        "preference.clipboard.content_settings.hints.display_lines",
      )}
      onChange={(value) => {
        clipboardStore.content.displayLines = value;
      }}
      options={options}
      title={t("preference.clipboard.content_settings.label.display_lines")}
      value={content.displayLines}
    />
  );
};

export default DisplayLines;
