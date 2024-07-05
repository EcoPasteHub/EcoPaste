import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryItem } from "@/types/database";
import { Flex, Tag } from "antd";
import clsx from "clsx";
import { last } from "lodash-es";
import { MacScrollbar } from "mac-scrollbar";

interface TabItem {
	label: string;
	value?: HistoryItem["group"];
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
		<MacScrollbar>
			<Flex data-tauri-drag-region>
				{tabList.map((item) => {
					const { label, value } = item;

					const isChecked = checked === label;

					return (
						<Tag.CheckableTag
							key={label}
							checked={isChecked}
							className={clsx({ "bg-primary!": isChecked })}
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
		</MacScrollbar>
	);
};

export default Tab;
