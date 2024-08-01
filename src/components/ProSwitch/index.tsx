import { Switch, type SwitchProps } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import type { FC } from "react";
import ProListItem from "../ProListItem";

type ProSwitchProps = SwitchProps & ListItemMetaProps;

const ProSwitch: FC<ProSwitchProps> = (props) => {
	const { title, description, children, ...rest } = props;

	return (
		<ProListItem title={title} description={description}>
			<Switch {...rest} />

			{children}
		</ProListItem>
	);
};

export default ProSwitch;
