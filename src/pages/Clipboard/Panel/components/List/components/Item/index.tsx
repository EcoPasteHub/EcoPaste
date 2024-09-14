import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import type { ClipboardItem } from "@/types/database";
import { copyFile, writeFile } from "@tauri-apps/api/fs";
import { downloadDir } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/api/shell";
import { Flex, type FlexProps } from "antd";
import clsx from "clsx";
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
}

interface MenuItem extends ContextMenu.Item {
	hide?: boolean;
}

const Item: FC<ItemProps> = (props) => {
	const { index, data, className, ...rest } = props;
	const { id, type, value, search, group, favorite, createTime } = data;
	const { state, getClipboardList } = useContext(ClipboardPanelContext);
	const { t } = useTranslation();
	const { env, appearance } = useSnapshot(globalStore);
	const { content } = useSnapshot(clipboardStore);

	state.$eventBus?.useSubscription((key) => {
		switch (key) {
			case LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW:
				return preview();
			case LISTEN_KEY.CLIPBOARD_ITEM_PASTE:
				return pasteValue();
			case LISTEN_KEY.CLIPBOARD_ITEM_DELETE:
				return deleteItem();
		}
	});

	const copy = () => {
		switch (type) {
			case "text":
				return writeText(value);
			case "rtf":
				return writeRTF(search, value);
			case "html":
				return writeHTML(search, value);
			case "image":
				return writeImage(value);
			case "files":
				return writeFiles(JSON.parse(value));
		}
	};

	const pastePlainText = async () => {
		await writeText(search);

		paste();
	};

	const toggleFavorite = async () => {
		await updateSQL("history", { id, favorite: !favorite });

		getClipboardList?.();
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
		const fileName = `${env.appName}_${id}.${ext}`;
		const destination = (await downloadDir()) + fileName;

		await writeFile(destination, value);

		previewPath(destination);
	};

	const preview = () => {
		if (state.activeId !== id || type !== "image") return;

		previewPath(value, false);
	};

	const downloadImage = async () => {
		const fileName = `${env.appName}_${id}.png`;
		const destination = (await downloadDir()) + fileName;

		await copyFile(value, destination);

		previewPath(destination);
	};

	const openFinder = () => {
		const [file] = JSON.parse(value);

		previewPath(file);
	};

	const deleteItem = async () => {
		if (state.activeId !== id) return;

		await deleteSQL("history", id);

		getClipboardList?.();
	};

	const deleteAbove = async () => {
		const list = state.data.list.filter((item) => {
			const isMore = item.createTime > createTime;
			const isDifferent = item.createTime === createTime && item.id !== id;

			return isMore || isDifferent;
		});

		await deleteAll(list);

		state.scrollToIndex?.(0);
	};

	const deleteBelow = async () => {
		const list = state.data.list.filter((item) => {
			const isLess = item.createTime < createTime;
			const isDifferent = item.createTime === createTime && item.id !== id;

			return isLess || isDifferent;
		});

		deleteAll(list);
	};

	const deleteOther = async () => {
		const list = state.data.list.filter((item) => item.id !== id);

		deleteAll(list);
	};

	const deleteAll = async (list: ClipboardItem[]) => {
		let filteredList = list;

		if (!state.favorite) {
			filteredList = list.filter((item) => !item.favorite);
		}

		for await (const item of filteredList) {
			await deleteSQL("history", item.id);
		}

		getClipboardList?.();
	};

	const pasteValue = async () => {
		if (state.activeId !== id) return;

		await copy();

		paste();
	};

	const handleContextMenu = async (event: MouseEvent) => {
		event.preventDefault();

		state.activeId = id;

		const menus: MenuItem[] = [
			{
				label: t("clipboard.button.context_menu.copy"),
				event: copy,
			},
			{
				label: t("clipboard.button.context_menu.paste_ocr_text"),
				hide: type !== "image" || /^[\s]*$/.test(search),
				event: pastePlainText,
			},
			{
				label: t("clipboard.button.context_menu.paste_as_plain_text"),
				hide: type !== "html" && type !== "rtf",
				event: pastePlainText,
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
			{
				label: t("clipboard.button.context_menu.delete_above"),
				hide: index === 0,
				event: deleteAbove,
			},
			{
				label: t("clipboard.button.context_menu.delete_below"),
				hide: index === state.data.list.length - 1,
				event: deleteBelow,
			},
			{
				label: t("clipboard.button.context_menu.delete_other"),
				hide: state.data.list.length === 1,
				event: deleteOther,
			},
			{
				label: t("clipboard.button.context_menu.delete_all"),
				hide: state.data.list.length === 1,
				event: () => deleteAll(state.data.list),
			},
		];

		showMenu({
			items: menus.filter(({ hide }) => !hide),
			// @ts-ignore
			theme: appearance.theme,
		});
	};

	const handleClick = (type: typeof content.autoPaste) => {
		state.activeId = id;

		if (content.autoPaste !== type) return;

		pasteValue();
	};

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
				{...data}
				copy={copy}
				toggleFavorite={toggleFavorite}
				deleteItem={deleteItem}
			/>

			<div className="flex-1 select-auto overflow-hidden break-words">
				{renderContent()}
			</div>
		</Flex>
	);
};

export default Item;
