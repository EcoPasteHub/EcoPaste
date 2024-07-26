import type { Language as TypeLanguage } from "@/types/store";
import { Flex, Select } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: TypeLanguage;
}

const Language = () => {
	const { language } = useSnapshot(globalStore);
	const { t } = useTranslation();

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
		<Flex align="center">
			<span>{t("preference.settings.basic.label.language")}：</span>
			<Select
				value={language}
				options={options}
				onChange={(value) => {
					globalStore.language = value;
				}}
			/>
		</Flex>
	);
};

export default Language;
