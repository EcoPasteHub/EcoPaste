import ProSelect from "@/components/ProSelect";
import type { Theme } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: Theme;
}

const ThemeMode = () => {
	const { appearance } = useSnapshot(globalStore);

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
			value={appearance.theme}
			options={options}
			onChange={(value) => {
				globalStore.appearance.theme = value;
			}}
		/>
	);
};

export default ThemeMode;
