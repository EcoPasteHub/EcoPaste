import type { ClipboardStore } from "@/types/store";
import { List, Segmented } from "antd";
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
		<List.Item
			actions={[
				<Segmented
					key={1}
					value={searchPosition}
					options={options}
					onChange={(value) => {
						clipboardStore.searchPosition = value;
					}}
				/>,
			]}
		>
			<List.Item.Meta title="搜索框位置" />
		</List.Item>
	);
};

export default SearchPosition;
