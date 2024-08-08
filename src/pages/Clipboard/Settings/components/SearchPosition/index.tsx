import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["search"]["position"];
}

const SearchPosition = () => {
	const { search } = useSnapshot(clipboardStore);

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
			value={search.position}
			options={options}
			onChange={(value) => {
				clipboardStore.search.position = value;
			}}
		/>
	);
};

export default SearchPosition;
