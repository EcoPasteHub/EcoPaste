import Scrollbar from "@/components/Scrollbar";
import type { HistoryTablePayload } from "@/types/database";
import { Flex, Tag } from "antd";
import clsx from "clsx";
import { MainContext } from "../..";

interface GroupItem extends Partial<HistoryTablePayload> {
	key: string;
	label: string;
}

const Group = () => {
	const { state } = useContext(MainContext);
	const { t } = useTranslation();
	const [checked, setChecked] = useState("all");

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

	useTauriFocus({
		onFocus() {
			if (!clipboardStore.window.showAll) return;

			handleChange(groupList[0]);
		},
	});

	useKeyPress("tab", (event) => {
		const index = groupList.findIndex((item) => item.key === checked);
		const length = groupList.length;

		let nextIndex = index;

		if (event.shiftKey) {
			nextIndex = index === 0 ? length - 1 : index - 1;
		} else {
			nextIndex = index === length - 1 ? 0 : index + 1;
		}

		handleChange(groupList[nextIndex]);
	});

	const handleChange = (item: GroupItem) => {
		const { key, group, favorite } = item;

		setChecked(key);

		// 切换时清空多选
		state.selectedIds = [];
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
