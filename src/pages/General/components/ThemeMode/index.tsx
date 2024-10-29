import ProSelect from "@/components/ProSelect";
import type { Theme } from "@/types/store";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { appearance } = useSnapshot(globalStore);
	const { t } = useTranslation();

	useMount(() => {
		const appWindow = getCurrentWebviewWindow();

		subscribeKey(
			globalStore.appearance,
			"theme",
			async (value) => {
				let nextTheme = value === "auto" ? null : value;

				await appWindow.setTheme(nextTheme);

				nextTheme = nextTheme ?? (await appWindow.theme());

				globalStore.appearance.isDark = nextTheme === "dark";
			},
			true,
		);

		// 监听系统主题的变化
		appWindow.onThemeChanged(async ({ payload }) => {
			if (globalStore.appearance.theme !== "auto") return;

			globalStore.appearance.isDark = payload === "dark";
		});
	});

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
			value={appearance.theme}
			options={options}
			onChange={(value) => {
				globalStore.appearance.theme = value;
			}}
		/>
	);
};

export default ThemeMode;
