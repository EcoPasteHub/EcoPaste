import Scrollbar from "@/components/Scrollbar";
import type { ClipboardItem } from "@/types/database";
import { Flex, Tag } from "antd";
import clsx from "clsx";
import { ClipboardPanelContext } from "../..";

interface GroupItem extends Partial<ClipboardItem> {
	key: string;
	label: string;
}

const Group = () => {
	const { state } = useContext(ClipboardPanelContext);
	const { t } = useTranslation();

	const groupList: GroupItem[] = [
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
			key: "favorite",
			label: t("clipboard.label.tab.favorite"),
			favorite: true,
		},
	];

	const [checked, setChecked] = useState(groupList[0].key);

	const handleChange = (item: GroupItem) => {
		const { key, group, favorite } = item;

		setChecked(key);

		Object.assign(state, { group, favorite });
	};

	return (
		<Scrollbar thumbSize={0}>
			<Flex data-tauri-drag-region>
				{groupList.map((item) => {
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

export default Group;
