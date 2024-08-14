import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["content"]["autoPaste"];
}

const AutoPaste = () => {
	const { content } = useSnapshot(clipboardStore);

	const options: Option[] = [
		{
			label: "关闭",
			value: "close",
		},
		{
			label: "单击",
			value: "single",
		},
		{
			label: "双击",
			value: "double",
		},
	];

	return (
		<ProSelect
			title="自动粘贴"
			description="鼠标左键的操作"
			value={content.autoPaste}
			options={options}
			onChange={(value) => {
				clipboardStore.content.autoPaste = value;
			}}
		/>
	);
};

export default AutoPaste;
