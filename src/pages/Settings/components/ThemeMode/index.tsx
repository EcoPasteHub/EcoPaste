import type { Theme } from "@/types/store";
import { Flex, Segmented } from "antd";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { theme, toggleTheme } = useTheme();
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.settings.basic.label.theme_mode_auto"),
			value: "auto",
		},
		{
			label: t("preference.settings.basic.label.theme_mode_light"),
			value: "light",
		},
		{
			label: t("preference.settings.basic.label.theme_mode_dark"),
			value: "dark",
		},
	];

	return (
		<Flex align="center">
			<span>{t("preference.settings.basic.label.theme_mode")}：</span>
			<Segmented value={theme} options={options} onChange={toggleTheme} />
		</Flex>
	);
};

export default ThemeMode;
