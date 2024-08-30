import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useSnapshot } from "valtio";
import Language from "./components/Language";
import MacosPermissions from "./components/MacosPermissions";
import ThemeMode from "./components/ThemeMode";

const General = () => {
	const { app } = useSnapshot(globalStore);
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
					title={t("preference.settings.app_settings.label.auto_update")}
					value={app.autoUpdate}
					onChange={(value) => {
						globalStore.app.autoUpdate = value;
					}}
				/>

				<ProSwitch
					title={"隐藏托盘图标"}
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
		</>
	);
};

export default General;
