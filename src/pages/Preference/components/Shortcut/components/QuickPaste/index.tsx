import { Select, Space, Switch } from "antd";
import type { DefaultOptionType } from "antd/es/select";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProListItem from "@/components/ProListItem";
import { modifierKeys } from "@/components/ProShortcut/keyboard";
import { globalStore } from "@/stores/global";

const QuickPaste = () => {
  const { shortcut } = useSnapshot(globalStore);
  const { t } = useTranslation();

  const options: DefaultOptionType[] = modifierKeys.map((item) => {
    const { tauriKey, symbol } = item;

    return {
      disabled: globalStore.shortcut.quickPaste.value === tauriKey,
      label: symbol,
      value: tauriKey,
    };
  });

  return (
    <ProListItem
      description={t("preference.shortcut.shortcut.hints.quick_paste")}
      title={t("preference.shortcut.shortcut.label.quick_paste")}
    >
      <Switch
        onChange={(value) => {
          globalStore.shortcut.quickPaste.enable = value;
        }}
        value={shortcut.quickPaste.enable}
      />

      <Space>
        <Select
          disabled={!shortcut.quickPaste.enable}
          maxCount={2}
          mode="multiple"
          onChange={(value) => {
            globalStore.shortcut.quickPaste.value = value.join("+");
          }}
          options={options}
          showSearch={false}
          value={shortcut.quickPaste.value?.split("+")}
        />

        <span>1~9</span>
      </Space>
    </ProListItem>
  );
};

export default QuickPaste;
