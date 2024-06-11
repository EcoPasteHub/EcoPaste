import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryGroup } from "@/types/database";
import { Flex, Tag } from "antd";
import type { FlexProps } from "antd/lib";
import { last } from "lodash-es";
import type { FC } from "react";

interface TabItem {
	label: string;
	value?: HistoryGroup;
}

const tabList: TabItem[] = [
	{
		label: "全部",
	},
	{
		label: "文本",
		value: "text",
	},
	{
		label: "图片",
		value: "image",
	},
	{
		label: "文件",
		value: "files",
	},
	{
		label: "收藏",
	},
];

const Tab: FC<Partial<FlexProps>> = (props) => {
	const { state } = useContext(HistoryContext);

	const [checked, setChecked] = useState(tabList[0].label);

	return (
		<Flex data-tauri-drag-region {...props}>
			{tabList.map((item) => {
				const { label, value } = item;

				return (
					<Tag.CheckableTag
						key={label}
						checked={checked === label}
						onChange={() => {
							setChecked(label);
							state.group = value;
							state.isCollected = label === last(tabList)?.label || undefined;
						}}
					>
						{label}
					</Tag.CheckableTag>
				);
			})}
		</Flex>
	);
};

export default Tab;
