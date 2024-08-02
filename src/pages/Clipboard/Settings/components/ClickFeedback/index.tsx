import ProSelect from "@/components/ProSelect";
import type { ClickFeedback as TypeClickFeedback } from "@/types/store";
import type { FC } from "react";

interface ClickFeedbackProps {
	label: string;
	value: TypeClickFeedback;
	onChange: (value: TypeClickFeedback) => void;
}

interface Option {
	label: string;
	value: TypeClickFeedback;
}

const ClickFeedback: FC<ClickFeedbackProps> = (props) => {
	const { label, ...rest } = props;

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

	return <ProSelect title={label} options={options} {...rest} />;
};

export default ClickFeedback;
