import { InputNumber } from "antd";
import { useSnapshot } from "valtio";
import ProListItem from "@/components/ProListItem";

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
