import { List } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import { Children, type FC } from "react";

const ProListItem: FC<ListItemMetaProps> = (props) => {
	const { title, description, children } = props;

	return (
		<List.Item actions={Children.toArray(children)}>
			<List.Item.Meta title={title} description={description} />
		</List.Item>
	);
};

export default ProListItem;
