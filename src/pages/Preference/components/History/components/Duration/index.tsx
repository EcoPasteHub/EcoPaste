import { InputNumber } from "antd";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProListItem from "@/components/ProListItem";
import { clipboardStore } from "@/stores/clipboard";

const Duration = () => {
  const { history } = useSnapshot(clipboardStore);
  const { t } = useTranslation();

  return (
    <ProListItem
      description={t("preference.history.history.hints.duration")}
      title={t("preference.history.history.label.duration")}
    >
      <InputNumber
        addonAfter={t("preference.history.history.label.duration_unit")}
        className="w-30"
        min={0}
        onChange={(value) => {
          clipboardStore.history.duration = value ?? 0;
        }}
        value={history.duration}
      />
    </ProListItem>
  );
};

export default Duration;
