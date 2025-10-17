import ProList from "@/components/ProList";
import ProShortcut from "@/components/ProShortcut";
import { useSnapshot } from "valtio";
import Preset from "./components/Preset";
import QuickPaste from "./components/QuickPaste";

const Shortcut = () => {
	const { shortcut } = useSnapshot(globalStore);
	const { t } = useTranslation();

	return (
		<>
			<ProList header={t("preference.shortcut.shortcut.title")}>
				<ProShortcut
					title={t("preference.shortcut.shortcut.label.open_clipboard")}
					value={shortcut.clipboard}
					onChange={(value) => {
						globalStore.shortcut.clipboard = value;
					}}
				/>

				<ProShortcut
					title={t("preference.shortcut.shortcut.label.open_settings")}
					value={shortcut.preference}
					onChange={(value) => {
						globalStore.shortcut.preference = value;
					}}
				/>

				<ProShortcut
					title={t("preference.shortcut.shortcut.label.open_notes")}
					value={shortcut.notes}
					onChange={(value) => {
						globalStore.shortcut.notes = value;
					}}
				/>

				<QuickPaste />

				<ProShortcut
					isSystem={false}
					title={t("preference.shortcut.shortcut.label.paste_as_plain")}
					description={t("preference.shortcut.shortcut.hints.paste_as_plain")}
					value={shortcut.pastePlain}
					onChange={(value) => {
						globalStore.shortcut.pastePlain = value;
					}}
				/>
			</ProList>

			<Preset />
		</>
	);
};

export default Shortcut;
