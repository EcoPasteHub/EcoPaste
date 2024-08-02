import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
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

	return (
		<ProSelect
			title="窗口位置"
			value={windowPosition}
			options={options}
			onChange={(value) => {
				clipboardStore.windowPosition = value;
			}}
		/>
	);
};

export default WindowPosition;
