import { Select, type SelectProps } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import ProListItem from "../ProListItem";

export type ProSelectProps<T> = SelectProps<T> & ListItemMetaProps;

const ProSelect = <T,>(props: ProSelectProps<T>) => {
	const { title, description, children, ...rest } = props;

	const width = useCreation(() => {
		let width = 0;

		const canvas = document.createElement("canvas");
		const context = canvas.getContext("2d");

		if (!context) return 100;

		context.font = getComputedStyle(document.body).font;

		for (const option of rest.options!) {
			const textWidth = context.measureText(option.label as string).width;

			if (textWidth > width) {
				width = textWidth;
			}
		}

		canvas.remove();

		return width + 43;
	}, [rest.options]);

	return (
		<ProListItem title={title} description={description}>
			<Select {...rest} style={{ width }} />

			{children}
		</ProListItem>
	);
};

export default ProSelect;
