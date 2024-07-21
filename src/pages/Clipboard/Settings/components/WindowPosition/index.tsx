import type { ClipboardStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["windowPosition"];
}

const WindowPosition = () => {
	const { windowPosition } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.clipboard.basic.label.window_position_default"),
			value: "default",
		},
		{
			label: t("preference.clipboard.basic.label.window_position_follow"),
			value: "follow",
		},
		{
			label: t("preference.clipboard.basic.label.window_position_center"),
			value: "center",
		},
	];

	const handleChange = (value: Option["value"]) => {
		clipboardStore.windowPosition = value;
	};

	return (
		<Flex align="center">
			{t("preference.clipboard.basic.label.window_position")}：
			<Segmented
				value={windowPosition}
				options={options}
				onChange={handleChange}
			/>
		</Flex>
	);
};

export default WindowPosition;
