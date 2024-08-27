import ProSelect from "@/components/ProSelect";
import type { Theme } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { appearance } = useSnapshot(globalStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.settings.appearance_settings.label.theme_auto"),
			value: "auto",
		},
		{
			label: t("preference.settings.appearance_settings.label.theme_light"),
			value: "light",
		},
		{
			label: t("preference.settings.appearance_settings.label.theme_dark"),
			value: "dark",
		},
	];

	return (
		<ProSelect
			title={t("preference.settings.appearance_settings.label.theme")}
			description={
				isWin() && t("preference.settings.appearance_settings.hints.theme")
			}
			value={appearance.theme}
			options={options}
			onChange={(value) => {
				globalStore.appearance.theme = value;
			}}
		/>
	);
};

export default ThemeMode;
