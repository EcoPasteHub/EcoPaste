import { HistoryContext } from "@/pages/Clipboard/History";
import type { HistoryItem } from "@/types/database";
import { BaseDirectory, copyFile, writeFile } from "@tauri-apps/api/fs";
import { open } from "@tauri-apps/api/shell";
import { Flex } from "antd";
import type { FC, KeyboardEvent, MouseEvent } from "react";
import { type ListChildComponentProps, areEqual } from "react-window";
import { type ContextMenu, showMenu } from "tauri-plugin-context-menu";
import { useSnapshot } from "valtio";
import Files from "./components/Files";
import HTML from "./components/HTML";
import Header from "./components/Header";
import Image from "./components/Image";
import RichText from "./components/RichText";
import Text from "./components/Text";

interface MenuItem extends ContextMenu.Item {
	hide?: boolean;
}

const Item: FC<ListChildComponentProps<HistoryItem[]>> = memo((props) => {
	const { index, style, data } = props;

	const {
		id,
		type,
		value = "",
		search = "",
		createTime = "",
		isCollected,
	} = data[index];

	const { appInfo } = useSnapshot(globalStore);
	const { visibleStartIndex, activeIndex } = useSnapshot(clipboardStore);
	const { getHistoryList } = useContext(HistoryContext);

	const containerRef = useRef<HTMLElement>(null);

	useEffect(() => {
		if (activeIndex === index) {
			containerRef.current?.focus();
		} else {
			containerRef.current?.blur();
		}
	}, [activeIndex]);

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
		const ext = type === "text" ? "txt" : type;

		writeFile(`${appInfo?.name}_${id}.${ext}`, value, {
			dir: BaseDirectory.Download,
		});
	};

	const previewImage = () => {
		previewFile(value, false);
	};

	const downloadImage = async () => {
		copyFile(value, `${appInfo?.name}_${id}.png`, {
			dir: BaseDirectory.Download,
		});
	};

	const openFinder = () => {
		const [file] = JSON.parse(value);

		previewFile(file);
	};

	const deleteItem = async () => {
		await deleteSQL("history", id);

		getHistoryList?.();
	};

	const deleteAbove = async () => {
		const aboveData = data.filter((item) => {
			const isMore = item.createTime! > createTime;
			const isDifferent = item.createTime === createTime && item.id !== id;

			return isMore || isDifferent;
		});

		for await (const item of aboveData) {
			await deleteSQL("history", item.id);
		}

		getHistoryList?.();
	};

	const deleteBelow = async () => {
		const belowData = data.filter((item) => {
			const isLess = item.createTime! < createTime;
			const isDifferent = item.createTime === createTime && item.id !== id;

			return isLess || isDifferent;
		});

		for await (const item of belowData) {
			await deleteSQL("history", item.id);
		}

		getHistoryList?.();
	};

	const deleteOther = async () => {
		const otherData = data.filter((item) => item.id !== id);

		for await (const item of otherData) {
			await deleteSQL("history", item.id);
		}

		getHistoryList?.();
	};

	const copyOCRText = async () => {
		writeText(search);
	};

	const handleContextMenu = async (event: MouseEvent) => {
		const { group } = data[index];

		event.preventDefault();

		const menus: MenuItem[] = [
			{
				label: "复制",
				event: copy,
			},
			{
				label: "复制OCR文本",
				hide: type !== "image",
				event: copyOCRText,
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
				label: (await isMac()) ? "在 Finder 中显示" : "在文件资源管理器中显示",
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
				hide: index === data.length - 1,
				event: deleteBelow,
			},
			{
				label: "删除其它",
				hide: data.length === 1,
				event: deleteOther,
			},
		];

		showMenu({ items: menus.filter(({ hide }) => !hide) });
	};

	const renderContent = () => {
		const props = data[index];

		switch (type) {
			case "rich-text":
				return <RichText {...props} />;
			case "html":
				return <HTML {...props} />;
			case "image":
				return <Image {...props} />;
			case "files":
				return <Files {...props} />;
			default:
				return <Text {...props} />;
		}
	};

	const handleFocus = () => {
		clipboardStore.activeIndex = index;
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

		if (isArrowDown && index < data.length - 1) {
			clipboardStore.activeIndex = index + 1;
		}
	};

	return (
		<Flex
			vertical
			ref={containerRef}
			tabIndex={0}
			gap={4}
			style={{
				...style,
				top: Number(style.top) + (index - visibleStartIndex) * 12,
			}}
			className="antd-input b-color-2 mx-12 h-full w-336! rounded-6 p-6"
			onContextMenu={handleContextMenu}
			onDoubleClick={copy}
			onFocus={handleFocus}
			onKeyDown={handleKeyDown}
		>
			<Header
				{...data[index]}
				copy={copy}
				collect={collect}
				deleteItem={deleteItem}
			/>

			<div className="flex-1 overflow-hidden">{renderContent()}</div>
		</Flex>
	);
}, areEqual);

export default Item;
