import Icon from "@/components/Icon";
import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import type { ClipboardItem } from "@/types/database";
import { Menu, MenuItem, type MenuItemOptions } from "@tauri-apps/api/menu";
import { downloadDir } from "@tauri-apps/api/path";
import { copyFile } from "@tauri-apps/plugin-fs";
import { open } from "@tauri-apps/plugin-shell";
import { Flex, type FlexProps } from "antd";
import clsx from "clsx";
import { find, isNil, remove } from "lodash-es";
import type { FC, MouseEvent } from "react";
import { useSnapshot } from "valtio";
import Files from "./components/Files";
import HTML from "./components/HTML";
import Header from "./components/Header";
import Image from "./components/Image";
import RTF from "./components/RTF";
import Text from "./components/Text";

interface ItemProps extends Partial<FlexProps> {
	index: number;
	data: ClipboardItem;
	openNoteModel: () => void;
}

interface ContextMenuItem extends MenuItemOptions {
	hide?: boolean;
}

const Item: FC<ItemProps> = (props) => {
	const { index, data, className, openNoteModel, ...rest } = props;
	const { id, type, value, search, group, favorite, note } = data;
	const { state } = useContext(ClipboardPanelContext);
	const { t } = useTranslation();
	const { env } = useSnapshot(globalStore);
	const { content } = useSnapshot(clipboardStore);

	state.$eventBus?.useSubscription((key) => {
		if (id !== state.eventBusId) return;

		switch (key) {
			case LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW:
				return preview();
			case LISTEN_KEY.CLIPBOARD_ITEM_PASTE:
				return pasteValue();
			case LISTEN_KEY.CLIPBOARD_ITEM_DELETE:
				return deleteItem();
			case LISTEN_KEY.CLIPBOARD_ITEM_SELECT_PREV:
				return selectNextOrPrev(false);
			case LISTEN_KEY.CLIPBOARD_ITEM_SELECT_NEXT:
				return selectNextOrPrev();
		}
	});

	// 复制
	const copy = () => {
		return writeClipboard(data);
	};

	// 粘贴纯文本
	const pastePlain = () => {
		pasteClipboard(data, true);
	};

	// 切换收藏状态
	const toggleFavorite = () => {
		const nextFavorite = !favorite;

		find(state.list, { id })!.favorite = nextFavorite;

		updateSQL("history", { id, favorite: nextFavorite });
	};

	// 打开链接至浏览器
	const openBrowser = () => {
		const url = value.startsWith("http") ? value : `http://${value}`;

		open(url);
	};

	// 发送邮件
	const sendEmail = () => {
		open(`mailto:${value}`);
	};

	// 导出文件
	const exportFile = async () => {
		const ext = type === "text" ? "txt" : type;
		const fileName = `${env.appName}_${id}.${ext}`;
		const destination = joinPath(await downloadDir(), fileName);

		await writeFile(destination, value);

		previewPath(destination);
	};

	// 预览
	const preview = () => {
		if (type !== "image") return;

		previewPath(value, false);
	};

	// 下载图片
	const downloadImage = async () => {
		const fileName = `${env.appName}_${id}.png`;
		const destination = joinPath(await downloadDir(), fileName);

		await copyFile(value, destination);

		previewPath(destination);
	};

	// 打开文件至访达
	const openFinder = () => {
		const [file] = JSON.parse(value);

		previewPath(file);
	};

	// 删除条目
	const deleteItem = () => {
		if (state.activeId === id) {
			const nextIndex = selectNextOrPrev();

			if (isNil(nextIndex)) {
				selectNextOrPrev(false);
			}
		}

		remove(state.list, { id });

		deleteSQL("history", id);
	};

	// 粘贴
	const pasteValue = () => {
		return pasteClipboard(data);
	};

	// 选中下一个或者上一个
	const selectNextOrPrev = (isNext = true) => {
		let nextIndex = index;

		if (isNext) {
			if (index === state.list.length - 1) return;

			nextIndex = index + 1;
		} else {
			if (index === 0) return;

			nextIndex = index - 1;
		}

		state.activeId = state.list[nextIndex]?.id;

		return nextIndex;
	};

	// 右键菜单
	const handleContextMenu = async (event: MouseEvent) => {
		event.preventDefault();

		state.activeId = id;

		const items: ContextMenuItem[] = [
			{
				text: t("clipboard.button.context_menu.copy"),
				action: copy,
			},
			{
				text: t("clipboard.button.context_menu.note"),
				action: openNoteModel,
			},
			{
				text: t("clipboard.button.context_menu.paste_ocr_text"),
				hide: type !== "image" || /^[\s]*$/.test(search),
				action: pastePlain,
			},
			{
				text: t("clipboard.button.context_menu.paste_as_plain_text"),
				hide: type !== "html" && type !== "rtf",
				action: pastePlain,
			},
			{
				text: favorite
					? t("clipboard.button.context_menu.unfavorite")
					: t("clipboard.button.context_menu.favorite"),
				action: toggleFavorite,
			},
			{
				text: t("clipboard.button.context_menu.open_in_browser"),
				hide: type !== "text" || !isURL(value),
				action: openBrowser,
			},
			{
				text: t("clipboard.button.context_menu.send_email"),
				hide: type !== "text" || !isEmail(value),
				action: sendEmail,
			},
			{
				text: t("clipboard.button.context_menu.export_as_file"),
				hide: group !== "text",
				action: exportFile,
			},
			{
				text: t("clipboard.button.context_menu.preview_image"),
				hide: type !== "image",
				action: preview,
			},
			{
				text: t("clipboard.button.context_menu.download_image"),
				hide: type !== "image",
				action: downloadImage,
			},
			{
				text: isMac()
					? t("clipboard.button.context_menu.show_in_finder")
					: t("clipboard.button.context_menu.show_in_file_explorer"),
				hide: type !== "files",
				action: openFinder,
			},
			{
				text: t("clipboard.button.context_menu.delete"),
				action: deleteItem,
			},
		];

		const menu = await Menu.new();

		for await (const item of items.filter(({ hide }) => !hide)) {
			const menuItem = await MenuItem.new(item);

			await menu.append(menuItem);
		}

		menu.popup();
	};

	// 点击事件
	const handleClick = (type: typeof content.autoPaste) => {
		state.activeId = id;

		if (content.autoPaste !== type) return;

		pasteValue();
	};

	// 渲染内容
	const renderContent = () => {
		switch (type) {
			case "rtf":
				return <RTF {...data} />;
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
			{...rest}
			vertical
			gap={4}
			className={clsx(
				className,
				"group antd-input! b-color-2 absolute inset-0 mx-12 h-full rounded-6 p-6",
				{
					"antd-input-focus!": state.activeId === id,
				},
			)}
			onContextMenu={handleContextMenu}
			onClick={() => handleClick("single")}
			onDoubleClick={() => handleClick("double")}
		>
			<Header
				data={data}
				copy={copy}
				toggleFavorite={toggleFavorite}
				deleteItem={deleteItem}
			/>

			<div className="relative flex-1 select-auto overflow-hidden break-words children:transition">
				<div
					className={clsx(
						"absolute inset-0 line-clamp-4 opacity-100 group-hover:opacity-0",
						{
							"opacity-0!": !note,
						},
					)}
				>
					<Icon
						name="i-hugeicons:task-edit-01"
						className="mr-2 translate-y-2"
					/>

					{note}
				</div>

				<div
					className={clsx("h-full opacity-0 group-hover:opacity-100", {
						"opacity-100!": !note,
					})}
				>
					{renderContent()}
				</div>
			</div>
		</Flex>
	);
};

export default Item;
