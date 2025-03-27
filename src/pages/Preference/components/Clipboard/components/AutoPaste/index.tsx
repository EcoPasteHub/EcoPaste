import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["content"]["autoPaste"];
}

const AutoPaste = () => {
	const { content } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.clipboard.content_settings.label.auto_paste_single"),
			value: "single",
		},
		{
			label: t("preference.clipboard.content_settings.label.auto_paste_double"),
			value: "double",
		},
	];

	return (
		<ProSelect
			title={t("preference.clipboard.content_settings.label.auto_paste")}
			value={content.autoPaste}
			description={t("preference.clipboard.content_settings.hints.auto_paste")}
			options={options}
			onChange={(value) => {
				clipboardStore.content.autoPaste = value;
			}}
		/>
	);
};

export default AutoPaste;
