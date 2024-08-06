import ProList from "@/components/ProList";
import ProShortcut from "@/components/ProShortcut";

const Shortcut = () => {
	const { t } = useTranslation();

	return (
		<ProList header={t("preference.shortcut.title")}>
			<ProShortcut
				title={t("preference.shortcut.clipboard")}
				defaultValue={clipboardStore.wakeUpKey}
				onChange={(value) => {
					clipboardStore.wakeUpKey = value;
				}}
			/>

			<ProShortcut
				title={t("preference.shortcut.settings")}
				defaultValue={globalStore.wakeUpKey}
				onChange={(value) => {
					globalStore.wakeUpKey = value;
				}}
			/>
		</ProList>
	);
};

export default Shortcut;
