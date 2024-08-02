import { Select, type SelectProps } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import ProListItem from "../ProListItem";

type ProSelect<T> = SelectProps<T> & ListItemMetaProps;

const ProSelect = <T,>(props: ProSelect<T>) => {
	const { title, description, children, ...rest } = props;

	return (
		<ProListItem title={title} description={description}>
			<Select {...rest} />

			{children}
		</ProListItem>
	);
};

export default ProSelect;
