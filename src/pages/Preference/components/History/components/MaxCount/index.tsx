import { InputNumber } from "antd";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProListItem from "@/components/ProListItem";
import { clipboardStore } from "@/stores/clipboard";

const MaxCount = () => {
  const { history } = useSnapshot(clipboardStore);
  const { t } = useTranslation();

  return (
    <ProListItem
      description={t("preference.history.history.hints.max_count")}
      title={t("preference.history.history.label.max_count")}
    >
      <InputNumber
        addonAfter={t("preference.history.history.label.max_count_unit")}
        className="w-30"
        min={0}
        onChange={(value) => {
          clipboardStore.history.maxCount = value ?? 0;
        }}
        value={history.maxCount}
      />
    </ProListItem>
  );
};

export default MaxCount;
