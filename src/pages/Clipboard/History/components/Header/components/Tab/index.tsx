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

const Tab = () => {
	const { state } = useContext(HistoryContext);
	const { t, i18n } = useTranslation();

	const tabList = useCreation<TabItem[]>(
		() => [
			{
				label: t("clipboard.label.tab.all"),
			},
			{
				label: t("clipboard.label.tab.text"),
				group: "text",
			},
			{
				label: t("clipboard.label.tab.image"),
				group: "image",
			},
			{
				label: t("clipboard.label.tab.files"),
				group: "files",
			},
			{
				label: t("clipboard.label.tab.collection"),
				isCollected: true,
			},
		],
		[i18n.language],
	);

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
