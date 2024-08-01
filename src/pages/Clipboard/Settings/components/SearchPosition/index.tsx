import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
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

	return (
		<ProSelect
			title="搜索框位置"
			value={searchPosition}
			options={options}
			onChange={(value) => {
				clipboardStore.searchPosition = value;
			}}
		/>
	);
};

export default SearchPosition;
