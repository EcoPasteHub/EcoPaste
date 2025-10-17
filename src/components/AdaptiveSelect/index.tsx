import type { AdaptiveSelectProps } from "@/types/plugin";
import { Select } from "antd";

const DEFAULT_WIDTH = 100;

const AdaptiveSelect = <T,>(props: AdaptiveSelectProps<T>) => {
	const { style, expandWidth = 0, ...rest } = props;

	const width = useCreation(() => {
		if (!rest.options) return DEFAULT_WIDTH;

		let width = 0;
		if (props.expandWidth !== 0 && props.expandWidth !== undefined) {
			width = props.expandWidth;
		}

		const canvas = document.createElement("canvas");
		const context = canvas.getContext("2d");

		if (!context) return DEFAULT_WIDTH;

		context.font = getComputedStyle(document.body).font;

		for (const option of rest.options) {
			const textWidth = context.measureText(option.label as string).width;

			if (textWidth > width) {
				width = textWidth;
			}
		}

		canvas.remove();

		return width + 43;
	}, [rest.options]);

	return (
		<Select
			{...rest}
			style={{
				...style,
				width,
			}}
		/>
	);
};

export default AdaptiveSelect;
