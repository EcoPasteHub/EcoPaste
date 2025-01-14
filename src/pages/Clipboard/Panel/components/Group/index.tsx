import Scrollbar from "@/components/Scrollbar";
import type { HistoryTablePayload } from "@/types/database";
import { Flex, Tag } from "antd";
import clsx from "clsx";
import { last } from "lodash-es";
import { ClipboardPanelContext } from "../..";

interface GroupItem extends Partial<HistoryTablePayload> {
	key: string;
	label: string;
}

const Group = () => {
	const { state } = useContext(ClipboardPanelContext);
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

	useOSKeyPress("tab", () => {
		const index = groupList.findIndex((item) => item.key === checked);

		if (index === groupList.length - 1) {
			handleChange(groupList[0]);
		} else {
			handleChange(groupList[index + 1]);
		}
	});

	useOSKeyPress("shift.tab", () => {
		const index = groupList.findIndex((item) => item.key === checked);

		if (index === 0) {
			handleChange(last(groupList)!);
		} else {
			handleChange(groupList[index - 1]);
		}
	});

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
