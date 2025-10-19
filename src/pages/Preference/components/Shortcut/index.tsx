import { useSnapshot } from "valtio";
import ProList from "@/components/ProList";
import ProShortcut from "@/components/ProShortcut";
import Preset from "./components/Preset";
import QuickPaste from "./components/QuickPaste";

const Shortcut = () => {
  const { shortcut } = useSnapshot(globalStore);
  const { t } = useTranslation();

  return (
    <>
      <ProList header={t("preference.shortcut.shortcut.title")}>
        <ProShortcut
          onChange={(value) => {
            globalStore.shortcut.clipboard = value;
          }}
          title={t("preference.shortcut.shortcut.label.open_clipboard")}
          value={shortcut.clipboard}
        />

        <ProShortcut
          onChange={(value) => {
            globalStore.shortcut.preference = value;
          }}
          title={t("preference.shortcut.shortcut.label.open_settings")}
          value={shortcut.preference}
        />

        <QuickPaste />

        <ProShortcut
          description={t("preference.shortcut.shortcut.hints.paste_as_plain")}
          isSystem={false}
          onChange={(value) => {
            globalStore.shortcut.pastePlain = value;
          }}
          title={t("preference.shortcut.shortcut.label.paste_as_plain")}
          value={shortcut.pastePlain}
        />
      </ProList>

      <Preset />
    </>
  );
};

export default Shortcut;
