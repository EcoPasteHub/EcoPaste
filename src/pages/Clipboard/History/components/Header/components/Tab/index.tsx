import Scrollbar from "@/components/Scrollbar";
import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryItem } from "@/types/database";
import { Flex, Tag } from "antd";
import clsx from "clsx";

interface TabItem {
	key: string;
	label: string;
	group?: HistoryItem["group"];
	isCollected?: boolean;
}

const Tab = () => {
	const { state } = useContext(HistoryContext);
	const { t } = useTranslation();

	const tabList: TabItem[] = [
		{
			key: "all",
			label: t("clipboard.label.tab.all"),
		},
		{
			key: "text",
			label: t("clipboard.label.tab.text"),
			group: "text",
		},
		{
			key: "image",
			label: t("clipboard.label.tab.image"),
			group: "image",
		},
		{
			key: "file",
			label: t("clipboard.label.tab.files"),
			group: "files",
		},
		{
			key: "collect",
			label: t("clipboard.label.tab.collection"),
			isCollected: true,
		},
	];

	const [checked, setChecked] = useState(tabList[0].key);

	const handleChange = (item: TabItem) => {
		const { key, group, isCollected } = item;

		setChecked(key);

		Object.assign(state, { group, isCollected });
	};

	return (
		<Scrollbar thumbSize={0}>
			<Flex data-tauri-drag-region>
				{tabList.map((item) => {
					const { key, label } = item;

					const isChecked = checked === key;

					return (
						<Tag.CheckableTag
							key={key}
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
