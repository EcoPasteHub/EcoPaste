import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryGroup } from "@/types/database";
import { Flex, Tag } from "antd";
import { last } from "lodash-es";

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

const Tab = () => {
	const { state } = useContext(HistoryContext);

	const [checked, setChecked] = useState(tabList[0].label);

	return (
		<Flex data-tauri-drag-region>
			{tabList.map((item) => {
				const { label, value } = item;

				return (
					<Tag.CheckableTag
						key={label}
						checked={checked === label}
						onChange={() => {
							setChecked(label);
							state.group = value;
							state.isFavorite = label === last(tabList)?.label || undefined;
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
