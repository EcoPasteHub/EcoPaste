import type { ClipboardStore } from "@/types/store";
import { Select } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["doubleClickFeedback"];
}

const DoubleClickFeedback = () => {
	const { doubleClickFeedback } = useSnapshot(clipboardStore);

	const options: Option[] = [
		{
			label: "无反馈",
			value: "none",
		},
		{
			label: "复制内容",
			value: "copy",
		},
		{
			label: "粘贴内容",
			value: "paste",
		},
	];

	const handleChange = (value: Option["value"]) => {
		clipboardStore.doubleClickFeedback = value;
	};

	return (
		<Select
			value={doubleClickFeedback}
			options={options}
			onChange={handleChange}
		/>
	);
};

export default DoubleClickFeedback;
