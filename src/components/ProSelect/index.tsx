import { List, Select, type SelectProps } from "antd";

interface ProSelect<T> extends SelectProps<T> {
	title: string;
	description?: string;
}

const ProSelect = <T,>(props: ProSelect<T>) => {
	const { title, description, ...rest } = props;

	return (
		<List.Item actions={[<Select key={1} {...rest} />]}>
			<List.Item.Meta title={title} description={description} />
		</List.Item>
	);
};

export default ProSelect;
