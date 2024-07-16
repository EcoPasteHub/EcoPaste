import type { ClipboardStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["defaultFocus"];
}

const DefaultFocus = () => {
	const { defaultFocus } = useSnapshot(clipboardStore);

	const options: Option[] = [
		{
			label: "首项",
			value: "firstItem",
		},
		{
			label: "搜索",
			value: "search",
		},
	];

	const handleChange = (value: Option["value"]) => {
		clipboardStore.defaultFocus = value;
	};

	return (
		<Flex align="center">
			默认聚焦：
			<Segmented
				value={defaultFocus}
				options={options}
				onChange={handleChange}
			/>
		</Flex>
	);
};

export default DefaultFocus;
