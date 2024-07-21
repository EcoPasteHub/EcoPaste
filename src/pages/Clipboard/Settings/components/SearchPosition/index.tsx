import type { ClipboardStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["searchPosition"];
}

const SearchPosition = () => {
	const { searchPosition } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.clipboard.basic.label.search_position_top"),
			value: "top",
		},
		{
			label: t("preference.clipboard.basic.label.search_position_bottom"),
			value: "bottom",
		},
	];

	const handleChange = (value: Option["value"]) => {
		clipboardStore.searchPosition = value;
	};

	return (
		<Flex align="center">
			{t("preference.clipboard.basic.label.search_position")}：
			<Segmented
				value={searchPosition}
				options={options}
				onChange={handleChange}
			/>
		</Flex>
	);
};

export default SearchPosition;
