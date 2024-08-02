import ProSelect from "@/components/ProSelect";
import type { Theme } from "@/types/store";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { theme, toggleTheme } = useTheme();

	const options: Option[] = [
		{
			label: "跟随系统",
			value: "auto",
		},
		{
			label: "亮色模式",
			value: "light",
		},
		{
			label: "暗色模式",
			value: "dark",
		},
	];

	return (
		<ProSelect
			title="主题模式"
			value={theme}
			options={options}
			onChange={toggleTheme}
		/>
	);
};

export default ThemeMode;
