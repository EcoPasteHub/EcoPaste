import type { ClipboardStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["searchPosition"];
}

const SearchPosition = () => {
	const { searchPosition } = useSnapshot(clipboardStore);

	const options: Option[] = [
		{
			label: "顶部",
			value: "top",
		},
		{
			label: "底部",
			value: "bottom",
		},
	];

	const handleChange = (value: Option["value"]) => {
		clipboardStore.searchPosition = value;
	};

	return (
		<Flex align="center">
			搜索位置：
			<Segmented
				value={searchPosition}
				options={options}
				onChange={handleChange}
			/>
		</Flex>
	);
};

export default SearchPosition;
