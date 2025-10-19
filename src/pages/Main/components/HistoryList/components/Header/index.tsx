import Scrollbar from "@/components/Scrollbar";
import UnoIcon from "@/components/UnoIcon";
import { MainContext } from "@/pages/Main";
import { transferData } from "@/pages/Preference/components/Clipboard/components/OperationButton";
import { pasteToClipboard, writeToClipboard } from "@/plugins/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";
import type { OperationButton } from "@/types/store";
import { Flex } from "antd";
import clsx from "clsx";
import { filesize } from "filesize";
import type { FC, MouseEvent } from "react";
import { useSnapshot } from "valtio";

interface HeaderProps {
	data: DatabaseSchemaHistory;
	handleNote: () => void;
	handleFavorite: () => void;
	handleDelete: () => void;
}

const Header: FC<HeaderProps> = (props) => {
	const { data } = props;
	const { id, type, value, count, createTime, favorite, subtype } = data;
	const { rootState } = useContext(MainContext);
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
					replace: [value.length],
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
		const { handleNote, handleFavorite, handleDelete } = props;

		event.stopPropagation();

		rootState.activeId = id;

		switch (key) {
			case "copy":
				return writeToClipboard(data);
			case "pastePlain":
				return pasteToClipboard(data, true);
			case "note":
				return handleNote();
			case "star":
				return handleFavorite();
			case "delete":
				return handleDelete();
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
					"opacity-100": rootState.activeId === id,
				})}
				onDoubleClick={(event) => event.stopPropagation()}
			>
				{operationButtons.map((item) => {
					const { key, icon, activeIcon, title } = item;

					const isFavorite = key === "star" && favorite;

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
