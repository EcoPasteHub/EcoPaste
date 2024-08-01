import { List, Switch, type SwitchProps } from "antd";
import type { FC } from "react";

interface ProSwitchProps extends SwitchProps {
	title: string;
	description?: string;
}

const ProSwitch: FC<ProSwitchProps> = (props) => {
	const { title, description, ...rest } = props;

	return (
		<List.Item actions={[<Switch key={1} {...rest} />]}>
			<List.Item.Meta title={title} description={description} />
		</List.Item>
	);
};

export default ProSwitch;
