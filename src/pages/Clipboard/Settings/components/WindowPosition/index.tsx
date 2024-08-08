import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["window"]["position"];
}

const WindowPosition = () => {
	const { window } = useSnapshot(clipboardStore);

	const options: Option[] = [
		{
			label: "记住位置",
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
			value={window.position}
			options={options}
			onChange={(value) => {
				clipboardStore.window.position = value;
			}}
		/>
	);
};

export default WindowPosition;
