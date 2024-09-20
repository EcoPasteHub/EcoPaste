import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useSnapshot } from "valtio";
import Language from "./components/Language";
import MacosPermissions from "./components/MacosPermissions";
import ThemeMode from "./components/ThemeMode";

const General = () => {
	const { app, update } = useSnapshot(globalStore);
	const { t } = useTranslation();

	return (
		<>
			<MacosPermissions />

			<ProList header={t("preference.settings.app_settings.title")}>
				<ProSwitch
					title={t("preference.settings.app_settings.label.auto_start")}
					value={app.autoStart}
					onChange={(value) => {
						globalStore.app.autoStart = value;
					}}
				/>

				<ProSwitch
					title={t("preference.settings.app_settings.label.hide_tray")}
					value={app.hideTray}
					onChange={(value) => {
						globalStore.app.hideTray = value;
					}}
				/>
			</ProList>

			<ProList header={t("preference.settings.appearance_settings.title")}>
				<Language />

				<ThemeMode />
			</ProList>

			<ProList header={t("preference.settings.app_update.title")}>
				<ProSwitch
					title={t("preference.settings.app_update.label.auto_update")}
					value={update.auto}
					onChange={(value) => {
						globalStore.update.auto = value;
					}}
				/>

				<ProSwitch
					title={t("preference.settings.app_update.label.update_beta")}
					description={t("preference.settings.app_update.hints.update_beta")}
					value={update.beta}
					onChange={(value) => {
						globalStore.update.beta = value;
					}}
				/>
			</ProList>
		</>
	);
};

export default General;
