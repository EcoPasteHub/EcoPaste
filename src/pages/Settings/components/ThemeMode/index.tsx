import type { Theme } from "@/types/store";
import { Flex, Segmented } from "antd";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { theme, toggleTheme } = useTheme();

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
