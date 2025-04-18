import type { SelectProps } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import AdaptiveSelect from "../AdaptiveSelect";
import ProListItem from "../ProListItem";

export type ProSelectProps<T> = SelectProps<T> & ListItemMetaProps;

const SettingSelect = <T,>(props: ProSelectProps<T>) => {
	const { title, description, children, ...rest } = props;

	return (
		<ProListItem title={title} description={description}>
			<AdaptiveSelect {...rest} />

			{children}
		</ProListItem>
	);
};

export default SettingSelect;
