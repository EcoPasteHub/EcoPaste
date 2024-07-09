import type { Theme } from "@/types/store";
import { appWindow } from "@tauri-apps/api/window";
import { Flex, Segmented } from "antd";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { theme, toggleTheme } = useTheme();

	useMount(() => {
		appWindow.onThemeChanged(({ payload }) => {
			if (globalStore.theme !== "auto") return;

			globalStore.isDark = payload === "dark";
		});
	});

	const options: Option[] = [
		{
			label: "自动",
			value: "auto",
		},
		{
			label: "亮色",
			value: "light",
		},
		{
			label: "暗色",
			value: "dark",
		},
	];

	return (
		<Flex align="center">
			<span>主题模式：</span>
			<Segmented value={theme} options={options} onChange={toggleTheme} />
		</Flex>
	);
};

export default ThemeMode;
