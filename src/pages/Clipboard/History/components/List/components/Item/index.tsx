import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryItem } from "@/types/database";
import { BaseDirectory, copyFile, writeFile } from "@tauri-apps/api/fs";
import { open } from "@tauri-apps/api/shell";
import { Flex } from "antd";
import type { FC, KeyboardEvent, MouseEvent } from "react";
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
}

interface MenuItem extends ContextMenu.Item {
	hide?: boolean;
}

const Item: FC<ItemProps> = (props) => {
	const { index, data } = props;
	const { id, type, group, value, search, createTime, isCollected } = data;

	const { state, getHistoryList } = useContext(HistoryContext);
	const { appInfo } = useSnapshot(globalStore);
	const { activeIndex, doubleClickFeedback } = useSnapshot(clipboardStore);

	const containerRef = useRef<HTMLElement>(null);

	useEffect(() => {
		if (activeIndex === index) {
			containerRef.current?.focus();
		} else {
			containerRef.current?.blur();
		}
	}, [activeIndex]);

	const copy = async () => {
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
		const ext = type === "text" ? "txt" : type;

		writeFile(`${appInfo?.name}_${id}.${ext}`, value, {
			dir: BaseDirectory.Download,
		});
	};

	const previewImage = async () => {
		previewPath(value, false);
	};

	const downloadImage = async () => {
		copyFile(value, `${appInfo?.name}_${id}.png`, {
			dir: BaseDirectory.Download,
		});
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
		for await (const item of list) {
			await deleteSQL("history", item.id);
		}

		getHistoryList?.();
	};

	const handleContextMenu = async (event: MouseEvent) => {
		event.preventDefault();

		const menus: MenuItem[] = [
			{
				label: "复制",
				event: copy,
			},
			{
				label: "复制OCR文本",
				hide: type !== "image",
				event: copyPlainText,
			},
			{
				label: "粘贴为纯文本",
				hide: type !== "html",
				event: copyPlainText,
			},
			{
				label: isCollected ? "取消收藏" : "收藏",
				event: collect,
			},
			{
				label: "在浏览器访问",
				hide: type !== "text" || !isURL(value),
				event: openBrowser,
			},
			{
				label: "发送邮件",
				hide: type !== "text" || !isEmail(value),
				event: sendEmail,
			},
			{
				label: "导出为文件",
				hide: group !== "text",
				event: exportFile,
			},
			{
				label: "预览图片",
				hide: type !== "image",
				event: previewImage,
			},
			{
				label: "下载图片",
				hide: type !== "image",
				event: downloadImage,
			},
			{
				label: isMac() ? "在 Finder 中显示" : "在文件资源管理器中显示",
				hide: type !== "files",
				event: openFinder,
			},
			{
				label: "删除",
				event: deleteItem,
			},
			{
				label: "删除上方",
				hide: index === 0,
				event: deleteAbove,
			},
			{
				label: "删除下方",
				hide: index === state.historyList.length - 1,
				event: deleteBelow,
			},
			{
				label: "删除其它",
				hide: state.historyList.length === 1,
				event: deleteOther,
			},
			{
				label: "删除所有",
				hide: state.historyList.length === 1,
				event: () => deleteAll(state.historyList),
			},
		];

		showMenu({ items: menus.filter(({ hide }) => !hide) });
	};

	const handleDoubleClick = () => {
		if (doubleClickFeedback === "none") return;

		if (doubleClickFeedback === "copy") {
			return copy();
		}
	};

	const handleFocus = () => {
		clipboardStore.activeIndex = index;
	};

	const handleBlur = () => {
		clipboardStore.activeIndex = -1;
	};

	const handleKeyDown = (event: KeyboardEvent) => {
		const isSpace = event.code === "Space";
		const isArrowUp = event.code === "ArrowUp";
		const isArrowDown = event.code === "ArrowDown";

		if (isSpace || isArrowUp || isArrowDown) {
			event.preventDefault();
		}

		if (isSpace && type === "image") {
			previewImage();
		}

		if (isArrowUp && index > 0) {
			clipboardStore.activeIndex = index - 1;
		}

		if (isArrowDown && index < state.historyList.length - 1) {
			clipboardStore.activeIndex = index + 1;
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
			className="antd-input b-color-2 mx-auto mb-12 h-120 w-336 rounded-6 p-6"
			onContextMenu={handleContextMenu}
			onDoubleClick={handleDoubleClick}
			onFocus={handleFocus}
			onBlur={handleBlur}
			onKeyDown={handleKeyDown}
		>
			<Header {...data} copy={copy} collect={collect} deleteItem={deleteItem} />

			<div className="flex-1 overflow-hidden">{renderContent()}</div>
		</Flex>
	);
};

export default memo(Item);
