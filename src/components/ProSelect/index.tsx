import type { SelectProps } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import EcoSelect from "../EcoSelect";
import ProListItem from "../ProListItem";

export type ProSelectProps<T> = SelectProps<T> & ListItemMetaProps;

const ProSelect = <T,>(props: ProSelectProps<T>) => {
	const { title, description, children, ...rest } = props;

	return (
		<ProListItem title={title} description={description}>
			<EcoSelect {...rest} />

			{children}
		</ProListItem>
	);
};

export default ProSelect;
