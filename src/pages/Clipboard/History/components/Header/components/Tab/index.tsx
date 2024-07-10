import Scrollbar from "@/components/Scrollbar";
import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryItem } from "@/types/database";
import { Flex, Tag } from "antd";
import clsx from "clsx";

interface TabItem {
	label: string;
	group?: HistoryItem["group"];
	isCollected?: boolean;
}

const tabList: TabItem[] = [
	{
		label: "全部",
	},
	{
		label: "文本",
		group: "text",
	},
	{
		label: "图片",
		group: "image",
	},
	{
		label: "文件",
		group: "files",
	},
	{
		label: "收藏",
		isCollected: true,
	},
];

const Tab = () => {
	const { state } = useContext(HistoryContext);

	const [checked, setChecked] = useState(tabList[0].label);

	const handleChange = (item: TabItem) => {
		const { label, group, isCollected } = item;

		setChecked(label);

		Object.assign(state, { group, isCollected });
	};

	return (
		<Scrollbar>
			<Flex data-tauri-drag-region>
				{tabList.map((item) => {
					const { label } = item;

					const isChecked = checked === label;

					return (
						<Tag.CheckableTag
							key={label}
							checked={isChecked}
							className={clsx({ "bg-primary!": isChecked })}
							onChange={() => handleChange(item)}
						>
							{label}
						</Tag.CheckableTag>
					);
				})}
			</Flex>
		</Scrollbar>
	);
};

export default Tab;
