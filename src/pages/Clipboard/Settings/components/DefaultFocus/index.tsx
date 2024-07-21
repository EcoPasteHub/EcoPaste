import type { ClipboardStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["defaultFocus"];
}

const DefaultFocus = () => {
	const { defaultFocus } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.clipboard.basic.label.default_focus_first_item"),
			value: "firstItem",
		},
		{
			label: t("preference.clipboard.basic.label.default_focus_search"),
			value: "search",
		},
	];

	const handleChange = (value: Option["value"]) => {
		clipboardStore.defaultFocus = value;
	};

	return (
		<Flex align="center">
			{t("preference.clipboard.basic.label.default_focus")}：
			<Segmented
				value={defaultFocus}
				options={options}
				onChange={handleChange}
			/>
		</Flex>
	);
};

export default DefaultFocus;
