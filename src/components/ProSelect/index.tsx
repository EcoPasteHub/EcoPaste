import { Select, type SelectProps } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import ProListItem from "../ProListItem";

export type ProSelectProps<T> = SelectProps<T> & ListItemMetaProps;

const ProSelect = <T,>(props: ProSelectProps<T>) => {
	const { title, description, children, ...rest } = props;

	return (
		<ProListItem title={title} description={description}>
			<Select {...rest} />

			{children}
		</ProListItem>
	);
};

export default ProSelect;
