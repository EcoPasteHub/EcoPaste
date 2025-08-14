import Scrollbar from "@/components/Scrollbar";
import UnoIcon from "@/components/UnoIcon";
import { MainContext } from "@/pages/Main";
import { transferData } from "@/pages/Preference/components/Clipboard/components/OperationButton";
import type { HistoryTablePayload } from "@/types/database";
import type { OperationButton } from "@/types/store";
import { Flex, Tooltip } from "antd";
import clsx from "clsx";
import { filesize } from "filesize";
import type { FC, MouseEvent } from "react";
import { useSnapshot } from "valtio";

interface HeaderProps {
	data: HistoryTablePayload;
	copy: () => void;
	pastePlain: () => void;
	openNoteModel: () => void;
	toggleFavorite: () => void;
	deleteItem: () => void;
}

const Header: FC<HeaderProps> = (props) => {
	const { data } = props;
	const { id, type, value, count, createTime, favorite, subtype } = data;
	const { state } = useContext(MainContext);
	const { t, i18n } = useTranslation();
	const { content } = useSnapshot(clipboardStore);

	const operationButtons = useCreation(() => {
		return content.operationButtons.map((key) => {
			return transferData.find((data) => data.key === key)!;
		});
	}, [content.operationButtons]);

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

	const handleClick = (event: MouseEvent, key: OperationButton) => {
		const { copy, pastePlain, openNoteModel, toggleFavorite, deleteItem } =
			props;

		event.stopPropagation();

		state.activeId = id;

		switch (key) {
			case "copy":
				return copy();
			case "pastePlain":
				return pastePlain();
			case "note":
				return openNoteModel();
			case "star":
				return toggleFavorite();
			case "delete":
				return deleteItem();
		}
	};

	return (
		<Flex justify="space-between" gap="small" className="text-color-2">
			<Scrollbar thumbSize={0}>
				<Flex gap="small" className="flex-1 whitespace-nowrap text-xs">
					<span>{renderType()}</span>
					<span>{renderCount()}</span>
					{renderPixel()}
					<span>{dayjs(createTime).locale(i18n.language).fromNow()}</span>
				</Flex>
			</Scrollbar>

			<Flex
				align="center"
				gap={6}
				className={clsx("opacity-0 transition group-hover:opacity-100", {
					"opacity-100": state.activeId === id,
				})}
				onDoubleClick={(event) => event.stopPropagation()}
			>
				{operationButtons.map((item) => {
					const { key, icon, activeIcon, title } = item;

					const isFavorite = key === "star" && favorite;

					if (key === "delete") {
						const [showConfirm, setShowConfirm] = useState(false);
						const [deleteConfirm, setDeleteConfirm] = useState(false);

						const handleDeleteAction = (event) => {
							event.stopPropagation();

							let shouldDelete = deleteConfirm;

							setShowConfirm(clipboardStore.content.deleteConfirm);

							if (!clipboardStore.content.deleteConfirm) {
								shouldDelete = true;
							}

							// 二次点击之后确认删除
							if (!deleteConfirm) {
								setDeleteConfirm(true);
							}

							if (shouldDelete) {
								handleClick(event, key);
							}
						};

						const cancelDelete = () => {
							setShowConfirm(false);
							setDeleteConfirm(false);
						};

						return (
							<Tooltip
								key={key}
								title={t("clipboard.hints.delete_modal_content")}
								placement={"left"}
								open={showConfirm}
								onMouseLeave={cancelDelete}
							>
								<UnoIcon
									hoverable
									name={icon}
									title={t(title)}
									className={clsx({ "text-red-5!": showConfirm })}
									onClick={(event) => handleDeleteAction(event)}
								/>
							</Tooltip>
						);
					}

					return (
						<UnoIcon
							key={key}
							hoverable
							name={isFavorite ? activeIcon : icon}
							title={t(title)}
							className={clsx({ "text-gold!": isFavorite })}
							onClick={(event) => handleClick(event, key)}
						/>
					);
				})}
			</Flex>
		</Flex>
	);
};

export default memo(Header);
