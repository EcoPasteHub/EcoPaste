import Icon from "@/components/Icon";
import Scrollbar from "@/components/Scrollbar";
import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import type { HistoryTablePayload } from "@/types/database";
import { Flex } from "antd";
import clsx from "clsx";
import { filesize } from "filesize";
import type { FC, MouseEvent } from "react";

interface HeaderProps {
	data: HistoryTablePayload;
	pastePlain: () => void;
	toggleFavorite: () => void;
	deleteItem: () => void;
}

const Header: FC<HeaderProps> = (props) => {
	const { data, pastePlain, toggleFavorite, deleteItem } = props;
	const { id, type, value, count, createTime, favorite, subtype } = data;
	const { state } = useContext(ClipboardPanelContext);
	const { t, i18n } = useTranslation();

	const renderType = () => {
		switch (subtype) {
			case "url":
				return t("clipboard.label.link");
			case "email":
				return t("clipboard.label.email");
			case "color":
				return t("clipboard.label.color");
			case "path":
				return t("clipboard.label.path");
		}

		switch (type) {
			case "text":
				return t("clipboard.label.plain_text");
			case "rtf":
				return t("clipboard.label.rtf");
			case "html":
				return t("clipboard.label.html");
			case "image":
				return t("clipboard.label.image");
			case "files":
				return t("clipboard.label.n_files", {
					replace: [JSON.parse(value).length],
				});
		}
	};

	const renderCount = () => {
		if (type === "files" || type === "image") {
			return filesize(count, { standard: "jedec" });
		}

		return t("clipboard.label.n_chars", {
			replace: [count],
		});
	};

	const renderPixel = () => {
		if (type !== "image") return;

		const { width, height } = data;

		return (
			<span>
				{width}×{height}
			</span>
		);
	};

	const handleClick = (event: MouseEvent) => {
		event.stopPropagation();

		state.activeId = id;
	};

	const handleDelete = (event: MouseEvent) => {
		event.stopPropagation();

		deleteItem();
	};

	return (
		<Flex justify="space-between" gap="small" className="color-2">
			<Scrollbar thumbSize={0}>
				<Flex gap="small" className="flex-1 whitespace-nowrap text-12">
					<span>{renderType()}</span>
					<span>{renderCount()}</span>
					{renderPixel()}
					<span>{dayjs(createTime).locale(i18n.language).fromNow()}</span>
				</Flex>
			</Scrollbar>

			<Flex
				align="center"
				gap={6}
				className={clsx(
					"text-14 opacity-0 transition group-hover:opacity-100",
					{
						"opacity-100": state.activeId === id,
					},
				)}
				onClick={handleClick}
				onDoubleClick={(event) => event.stopPropagation()}
			>
				<Icon hoverable name="i-lucide:clipboard-paste" onClick={pastePlain} />

				<Icon
					hoverable
					name={favorite ? "i-iconamoon:star-fill" : "i-iconamoon:star"}
					className={clsx({ "text-gold!": favorite })}
					onClick={toggleFavorite}
				/>

				<Icon
					hoverable
					size={15}
					name="i-iconamoon:trash-simple"
					className="hover:text-danger!"
					onClick={handleDelete}
				/>
			</Flex>
		</Flex>
	);
};

export default memo(Header);
