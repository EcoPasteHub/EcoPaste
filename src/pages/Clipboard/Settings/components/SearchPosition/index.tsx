import type { ClipboardStore } from "@/types/store";
import { Segmented } from "antd";
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
		<Segmented
			value={searchPosition}
			options={options}
			onChange={handleChange}
		/>
	);
};

export default SearchPosition;
