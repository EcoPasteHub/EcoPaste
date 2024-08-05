import ProSelect from "@/components/ProSelect";
import type { Language as TypeLanguage } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: TypeLanguage;
}

const Language = () => {
	const { language } = useSnapshot(globalStore);

	const options: Option[] = [
		{
			label: "简体中文",
			value: "zh-CN",
		},
		{
			label: "繁體中文",
			value: "zh-TW",
		},
		{
			label: "English",
			value: "en-US",
		},
		{
			label: "日本語",
			value: "ja-JP",
		},
	];

	return (
		<ProSelect
			title="界面语言"
			value={language}
			options={options}
			onChange={(value) => {
				globalStore.language = value;
			}}
		/>
	);
};

export default Language;
