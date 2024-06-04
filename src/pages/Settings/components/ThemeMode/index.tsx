import type { Theme } from "@/types/store";
import { invoke } from "@tauri-apps/api";
import { appWindow } from "@tauri-apps/api/window";
import { Segmented } from "antd";
import { useSnapshot } from "valtio";

const ThemeMode = () => {
	const { theme } = useSnapshot(store);

	useMount(() => {
		appWindow.onThemeChanged(({ payload }) => {
			if (store.theme !== "auto") return;

			store.isDark = payload === "dark";
		});
	});

	return (
		<Segmented
			value={theme}
			options={[
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
			]}
			onChange={(value: Theme) => {
				store.theme = value;

				invoke("plugin:theme|set_theme", {
					theme: value,
				});
			}}
		/>
	);
};

export default ThemeMode;
