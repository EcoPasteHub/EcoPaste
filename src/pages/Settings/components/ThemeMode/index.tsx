import type { Theme } from "@/types/store";
import { invoke } from "@tauri-apps/api";
import { appWindow } from "@tauri-apps/api/window";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { theme } = useSnapshot(store);

	useMount(() => {
		appWindow.onThemeChanged(({ payload }) => {
			if (store.theme !== "auto") return;

			store.isDark = payload === "dark";
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
			<Segmented
				value={theme}
				options={options}
				onChange={(value) => {
					store.theme = value;

					invoke("plugin:theme|set_theme", {
						theme: value,
					});
				}}
			/>
		</Flex>
	);
};

export default ThemeMode;
