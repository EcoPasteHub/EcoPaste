import ProSelect from "@/components/ProSelect";
import type { Language as TypeLanguage } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: TypeLanguage;
}

const Language = () => {
	const { appearance } = useSnapshot(globalStore);
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
		<ProSelect
			title={t("preference.settings.appearance_settings.label.language")}
			value={appearance.language}
			options={options}
			onChange={(value) => {
				globalStore.appearance.language = value;
			}}
		/>
	);
};

export default Language;
