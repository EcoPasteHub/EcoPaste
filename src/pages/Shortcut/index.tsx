import ProList from "@/components/ProList";
import ProShortcut from "@/components/ProShortcut";
import { useSnapshot } from "valtio";
import QuickPaste from "./components/QuickPaste";

const Shortcut = () => {
	const { shortcut } = useSnapshot(globalStore);
	const { t } = useTranslation();

	return (
		<ProList header={t("preference.shortcut.shortcut.title")}>
			<ProShortcut
				title={t("preference.shortcut.shortcut.label.open_clipboard")}
				defaultValue={shortcut.clipboard}
				onChange={(value) => {
					globalStore.shortcut.clipboard = value;
				}}
			/>

			<ProShortcut
				title={t("preference.shortcut.shortcut.label.open_settings")}
				defaultValue={shortcut.preference}
				onChange={(value) => {
					globalStore.shortcut.preference = value;
				}}
			/>

			<QuickPaste />
		</ProList>
	);
};

export default Shortcut;
