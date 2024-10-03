import Icon from "@/components/Icon";
import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import type { ClipboardItem } from "@/types/database";
import { copyFile, writeFile } from "@tauri-apps/api/fs";
import { downloadDir } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/api/shell";
import { Flex, type FlexProps } from "antd";
import clsx from "clsx";
import { find, isNil, remove } from "lodash-es";
import type { FC, MouseEvent } from "react";
import { type ContextMenu, showMenu } from "tauri-plugin-context-menu";
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
	openRemarkModel: () => void;
}

interface ContextMenuItem extends ContextMenu.Item {
	hide?: boolean;
}

const Item: FC<ItemProps> = (props) => {
	const { index, data, className, openRemarkModel, ...rest } = props;
	const { id, type, value, search, group, favorite, remark } = data;
	const { state } = useContext(ClipboardPanelContext);
	const { t } = useTranslation();
	const { env, appearance } = useSnapshot(globalStore);
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
	const openBrowser = async () => {
		const url = value.startsWith("http") ? value : `http://${value}`;

		open(url);
	};

	// 发送邮件
	const sendEmail = async () => {
		open(`mailto:${value}`);
	};

	// 导出文件
	const exportFile = async () => {
		const ext = type === "text" ? "txt" : type;
		const fileName = `${env.appName}_${id}.${ext}`;
		const destination = (await downloadDir()) + fileName;

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
		const destination = (await downloadDir()) + fileName;

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
	const pasteValue = async () => {
		pasteClipboard(data);
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

		const menus: ContextMenuItem[] = [
			{
				label: t("clipboard.button.context_menu.copy"),
				event: copy,
			},
			{
				label: "备注",
				event: openRemarkModel,
			},
			{
				label: t("clipboard.button.context_menu.paste_ocr_text"),
				hide: type !== "image" || /^[\s]*$/.test(search),
				event: pastePlain,
			},
			{
				label: t("clipboard.button.context_menu.paste_as_plain_text"),
				hide: type !== "html" && type !== "rtf",
				event: pastePlain,
			},
			{
				label: favorite
					? t("clipboard.button.context_menu.unfavorite")
					: t("clipboard.button.context_menu.favorite"),
				event: toggleFavorite,
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
				event: preview,
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
		];

		showMenu({
			items: menus.filter(({ hide }) => !hide),
			theme: appearance.theme as ContextMenu.Theme,
		});
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
				"group antd-input! b-color-2 absolute inset-0 mx-9 h-full rounded-6 p-6",
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
							"opacity-0!": !remark,
						},
					)}
				>
					<Icon
						name="i-hugeicons:task-edit-01"
						className="mr-2 translate-y-2"
					/>

					{remark}
				</div>

				<div
					className={clsx("h-full opacity-0 group-hover:opacity-100", {
						"opacity-100!": !remark,
					})}
				>
					{renderContent()}
				</div>
			</div>
		</Flex>
	);
};

export default Item;
