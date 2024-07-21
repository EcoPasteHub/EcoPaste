import type { ClipboardStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["doubleClickFeedback"];
}

const DoubleClickFeedback = () => {
	const { doubleClickFeedback } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.clipboard.basic.label.dbl_click_feedback_none"),
			value: "none",
		},
		{
			label: t("preference.clipboard.basic.label.dbl_click_feedback_copy"),
			value: "copy",
		},
		{
			label: t("preference.clipboard.basic.label.dbl_click_feedback_paste"),
			value: "paste",
		},
	];

	const handleChange = (value: Option["value"]) => {
		clipboardStore.doubleClickFeedback = value;
	};

	return (
		<Flex align="center">
			{t("preference.clipboard.basic.label.dbl_click_feedback")}：
			<Segmented
				value={doubleClickFeedback}
				options={options}
				onChange={handleChange}
			/>
		</Flex>
	);
};

export default DoubleClickFeedback;
