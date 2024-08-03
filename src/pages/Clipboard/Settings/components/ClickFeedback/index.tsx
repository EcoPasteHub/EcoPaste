import ProSelect from "@/components/ProSelect";
import type { ProSelectProps } from "@/components/ProSelect";
import type { ClickFeedback as TypeClickFeedback } from "@/types/store";

interface Option {
	label: string;
	value: TypeClickFeedback;
}

const ClickFeedback = <T,>(props: ProSelectProps<T>) => {
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

	return <ProSelect {...props} options={options} />;
};

export default ClickFeedback;
