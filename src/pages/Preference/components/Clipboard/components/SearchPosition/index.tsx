import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProSelect from "@/components/ProSelect";
import { clipboardStore } from "@/stores/clipboard";
import type { ClipboardStore } from "@/types/store";

interface Option {
  label: string;
  value: ClipboardStore["search"]["position"];
}

const SearchPosition = () => {
  const { search } = useSnapshot(clipboardStore);
  const { t } = useTranslation();

  const options: Option[] = [
    {
      label: t("preference.clipboard.search_box_settings.label.position_top"),
      value: "top",
    },
    {
      label: t(
        "preference.clipboard.search_box_settings.label.position_bottom",
      ),
      value: "bottom",
    },
  ];

  return (
    <ProSelect
      onChange={(value) => {
        clipboardStore.search.position = value;
      }}
      options={options}
      title={t("preference.clipboard.search_box_settings.label.position")}
      value={search.position}
    />
  );
};

export default SearchPosition;
