import type { ClipboardStore } from "@/types/store";
import { Select } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["windowPosition"];
}

const WindowPosition = () => {
	const { windowPosition } = useSnapshot(clipboardStore);

	const options: Option[] = [
		{
			label: "自由拖动",
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

	const handleChange = (value: Option["value"]) => {
		clipboardStore.windowPosition = value;
	};

	return (
		<Select value={windowPosition} options={options} onChange={handleChange} />
	);
};

export default WindowPosition;
