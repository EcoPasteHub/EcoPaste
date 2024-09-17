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

			<QuickPaste />

			<ProShortcut
				title="粘贴为纯文本"
				description="将内容粘贴为纯文本或 OCR 文本"
				value={shortcut.pastePlain}
				onChange={(value) => {
					globalStore.shortcut.pastePlain = value;
				}}
			/>
		</ProList>
	);
};

export default Shortcut;
