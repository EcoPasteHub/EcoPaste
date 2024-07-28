import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryItem } from "@/types/database";
import { copyFile, writeFile } from "@tauri-apps/api/fs";
import { downloadDir } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/api/shell";
import { Flex } from "antd";
import type { CSSProperties, FC, KeyboardEvent, MouseEvent } from "react";
import { type ContextMenu, showMenu } from "tauri-plugin-context-menu";
import { useSnapshot } from "valtio";
import Files from "./components/Files";
import HTML from "./components/HTML";
import Header from "./components/Header";
import Image from "./components/Image";
import RichText from "./components/RichText";
import Text from "./components/Text";

interface ItemProps {
	index: number;
	data: HistoryItem;
	style: CSSProperties;
}

interface MenuItem extends ContextMenu.Item {
	hide?: boolean;
}

const Item: FC<ItemProps> = (props) => {
	const { index, style, data } = props;
	const { id, type, group, value, search, createTime, isCollected } = data;

	const { state, getHistoryList } = useContext(HistoryContext);
	const { appInfo, theme } = useSnapshot(globalStore);
	const { doubleClickFeedback, clickPaste } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const containerRef = useRef<HTMLElement>(null);

	useEffect(() => {
		if (state.searching) return;

		if (state.activeIndex === index) {
			containerRef.current?.focus();
		} else {
			containerRef.current?.blur();
		}
	}, [state.activeIndex, state.searching, state.historyList]);

	const copy = () => {
		switch (type) {
			case "text":
				return writeText(value);
			case "rich-text":
				return writeRichText(value);
			case "html":
				return writeHTML(search, value);
			case "image":
				return writeImage(value);
			case "files":
				return writeFiles(JSON.parse(value));
		}
	};

	const copyPlainText = () => {
		writeText(search);
	};

	const collect = async () => {
		await updateSQL("history", { id, isCollected: !isCollected });

		getHistoryList?.();
	};

	const openBrowser = async () => {
		const url = value.startsWith("http") ? value : `http://${value}`;

		open(url);
	};

	const sendEmail = async () => {
		open(`mailto:${value}`);
	};

	const exportFile = async () => {
		const extension = type === "text" ? "txt" : type;
		const fileName = `${appInfo?.name}_${id}.${extension}`;
		const destination = (await downloadDir()) + fileName;

		await writeFile(destination, value);

		previewPath(destination);
	};

	const previewImage = async () => {
		previewPath(value, false);
	};

	const downloadImage = async () => {
		const fileName = `${appInfo?.name}_${id}.png`;
		const destination = (await downloadDir()) + fileName;

		await copyFile(value, destination);

		previewPath(destination);
	};

	const openFinder = () => {
		const [file] = JSON.parse(value);

		previewPath(file);
	};

	const deleteItem = async () => {
		await deleteSQL("history", id);

		getHistoryList?.();
	};

	const deleteAbove = async () => {
		const list = state.historyList.filter((item) => {
			const isMore = item.createTime > createTime;
			const isDifferent = item.createTime === createTime && item.id !== id;

			return isMore || isDifferent;
		});

		deleteAll(list);
	};

	const deleteBelow = async () => {
		const list = state.historyList.filter((item) => {
			const isLess = item.createTime < createTime;
			const isDifferent = item.createTime === createTime && item.id !== id;

			return isLess || isDifferent;
		});

		deleteAll(list);
	};

	const deleteOther = async () => {
		const list = state.historyList.filter((item) => item.id !== id);

		deleteAll(list);
	};

	const deleteAll = async (list: HistoryItem[]) => {
		let filteredList = list;

		if (!state.isCollected) {
			filteredList = list.filter((item) => !item.isCollected);
		}

		for await (const item of filteredList) {
			await deleteSQL("history", item.id);
		}

		getHistoryList?.();
	};

	const pasteValue = async () => {
		await copy();

		if (isMac()) {
			paste();
		} else {
			hideWindow();

			await paste();

			if (!state.pin) return;

			showWindow();
		}
	};

	const handleContextMenu = async (event: MouseEvent) => {
		event.preventDefault();

		const menus: MenuItem[] = [
			{
				label: t("clipboard.button.context_menu.copy"),
				event: copy,
			},
			{
				label: t("clipboard.button.context_menu.copy_ocr_text"),
				hide: type !== "image" || /^[\s]*$/.test(search),
				event: copyPlainText,
			},
			{
				label: t("clipboard.button.context_menu.paste_as_plain_text"),
				hide: type !== "html",
				event: copyPlainText,
			},
			{
				label: isCollected
					? t("clipboard.button.context_menu.unfavorite")
					: t("clipboard.button.context_menu.favorite"),
				event: collect,
			},
			{
				label: t("clipboard.button.context_menu.open_in_browser"),
				hide: type !== "text" || !isURL(value),
				event: openBrowser,
			},
			{
				label: t("clipboard.button.context_menu.send_email"),
				hide: type !== "text" || !isEmail(value),
				event: sendEmail,
			},
			{
				label: t("clipboard.button.context_menu.export_as_file"),
				hide: group !== "text",
				event: exportFile,
			},
			{
				label: t("clipboard.button.context_menu.preview_image"),
				hide: type !== "image",
				event: previewImage,
			},
			{
				label: t("clipboard.button.context_menu.download_image"),
				hide: type !== "image",
				event: downloadImage,
			},
			{
				label: isMac()
					? t("clipboard.button.context_menu.show_in_finder")
					: t("clipboard.button.context_menu.show_in_file_explorer"),
				hide: type !== "files",
				event: openFinder,
			},
			{
				label: t("clipboard.button.context_menu.delete"),
				event: deleteItem,
			},
			{
				label: t("clipboard.button.context_menu.delete_above"),
				hide: index === 0,
				event: deleteAbove,
			},
			{
				label: t("clipboard.button.context_menu.delete_below"),
				hide: index === state.historyList.length - 1,
				event: deleteBelow,
			},
			{
				label: t("clipboard.button.context_menu.delete_other"),
				hide: state.historyList.length === 1,
				event: deleteOther,
			},
			{
				label: t("clipboard.button.context_menu.delete_all"),
				hide: state.historyList.length === 1,
				event: () => deleteAll(state.historyList),
			},
		];

		showMenu({
			items: menus.filter(({ hide }) => !hide),
			// @ts-ignore
			theme,
		});
	};

	const handleClick = () => {
		if (!clickPaste) return;

		pasteValue();
	};

	const handleDoubleClick = () => {
		if (clickPaste || doubleClickFeedback === "none") return;

		if (doubleClickFeedback === "copy") {
			return copy();
		}

		pasteValue();
	};

	const handleFocus = () => {
		state.activeIndex = index;
	};

	const handleKeyDown = (event: KeyboardEvent) => {
		const isSpace = event.code === "Space";
		const isArrowUp = event.code === "ArrowUp";
		const isArrowDown = event.code === "ArrowDown";
		const isEnter = event.code === "Enter";

		if (isSpace || isArrowUp || isArrowDown || isEnter) {
			event.preventDefault();
		}

		if (isSpace && type === "image") {
			previewImage();
		}

		if (isArrowUp && index > 0) {
			state.activeIndex = index - 1;
		}

		if (isArrowDown && index < state.historyList.length - 1) {
			state.activeIndex = index + 1;
		}

		if (isEnter) {
			pasteValue();
		}
	};

	const renderContent = () => {
		switch (type) {
			case "rich-text":
				return <RichText {...data} />;
			case "html":
				return <HTML {...data} />;
			case "image":
				return <Image {...data} />;
			case "files":
				return <Files {...data} />;
			default:
				return <Text {...data} />;
		}
	};

	return (
		<Flex
			vertical
			ref={containerRef}
			tabIndex={0}
			gap={4}
			style={style}
			className="antd-input b-color-2 absolute inset-0 mx-12 h-full w-336! rounded-6 p-6"
			onContextMenu={handleContextMenu}
			onClick={handleClick}
			onDoubleClick={handleDoubleClick}
			onFocus={handleFocus}
			onKeyDown={handleKeyDown}
		>
			<Header {...data} copy={copy} collect={collect} deleteItem={deleteItem} />

			<div className="flex-1 overflow-hidden">{renderContent()}</div>
		</Flex>
	);
};

export default memo(Item);
