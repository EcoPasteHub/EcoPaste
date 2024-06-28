import type { WindowPosition as WindowPositionType } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: WindowPositionType;
}

const WindowPosition = () => {
	const { windowPosition } = useSnapshot(clipboardStore);

	const options: Option[] = [
		{
			label: "默认位置",
			value: "default",
		},
		{
			label: "跟随鼠标",
			value: "follow",
		},
		{
			label: "屏幕中心",
			value: "center",
		},
	];

	const handleChange = (value: WindowPositionType) => {
		clipboardStore.windowPosition = value;
	};

	return (
		<Flex align="center">
			窗口位置：
			<Segmented
				value={windowPosition}
				options={options}
				onChange={handleChange}
			/>
		</Flex>
	);
};

export default WindowPosition;
