import { Input } from "antd";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProListItem from "@/components/ProListItem";
import { globalStore } from "@/stores/global";

const TenorApiKey = () => {
  const { integration } = useSnapshot(globalStore);
  const { t } = useTranslation();

  return (
    <ProListItem
      description={t(
        "preference.settings.integration_settings.hints.tenor_api_key",
      )}
      title={t("preference.settings.integration_settings.label.tenor_api_key")}
    >
      <Input.Password
        className="w-55"
        onChange={(event) => {
          globalStore.integration.tenorApiKey = event.target.value.trim();
        }}
        placeholder={t("clipboard.gif.hints.api_placeholder")}
        value={integration.tenorApiKey}
      />
    </ProListItem>
  );
};

export default TenorApiKey;
