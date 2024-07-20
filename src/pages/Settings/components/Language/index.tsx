import type { GlobalStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: GlobalStore["language"];
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
			label: "English",
			value: "en-US",
		},
	];

	return (
		<Flex align="center">
			<span>{t("preference.settings.basic.label.language")}：</span>
			<Segmented
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
