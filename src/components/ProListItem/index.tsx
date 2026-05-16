import { List } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import { Children, type FC } from "react";

const ProListItem: FC<ListItemMetaProps> = (props) => {
  const { children, ...rest } = props;

  return (
    <List.Item actions={Children.toArray(children)}>
      <List.Item.Meta {...rest} />
    </List.Item>
  );
};

export default ProListItem;
