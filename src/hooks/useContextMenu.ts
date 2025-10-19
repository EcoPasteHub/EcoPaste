import { MainContext } from "@/pages/Main";
import type { ItemProps } from "@/pages/Main/components/HistoryList/components/Item";
import { writeToClipboard } from "@/plugins/clipboard";
import { Menu, MenuItem, type MenuItemOptions } from "@tauri-apps/api/menu";
import { downloadDir } from "@tauri-apps/api/path";
import { copyFile, writeTextFile } from "@tauri-apps/plugin-fs";
import { openUrl, revealItemInDir } from "@tauri-apps/plugin-opener";
import { find, isArray, remove } from "es-toolkit/compat";
import type { MouseEvent } from "react";
import { useSnapshot } from "valtio";

interface ContextMenuItem extends MenuItemOptions {
	hide?: boolean;
}

export const useContextMenu = (props: ItemProps) => {
	const { data, deleteModal, handleNote } = props;
	const { id, type, value, group, favorite, subtype } = data;
	const { t } = useTranslation();
	const { env } = useSnapshot(globalStore);
	const { rootState } = useContext(MainContext);

	const pasteAsText = () => {
		return pasteToClipboard(data, true);
	};

	const handleFavorite = async () => {
		const nextFavorite = !favorite;

		const matched = find(rootState.list, { id });

		if (!matched) return;

		matched.favorite = nextFavorite;

		const db = await getDatabase();

		await db
			.updateTable("history")
			.set("favorite", nextFavorite)
			.where("id", "=", id)
			.execute();
	};

	const openToBrowser = () => {
		if (type !== "text") return;

		const url = value.startsWith("http") ? value : `http://${value}`;

		openUrl(url);
	};

	const exportToFile = async () => {
		if (isArray(value)) return;

		const extname = type === "text" ? "txt" : type;
		const fileName = `${env.appName}_${id}.${extname}`;
		const path = join(await downloadDir(), fileName);

		await writeTextFile(path, value);

		revealItemInDir(path);
	};

	const downloadImage = async () => {
		if (type !== "image") return;

		const fileName = `${env.appName}_${id}.png`;
		const path = join(await downloadDir(), fileName);

		await copyFile(value, path);

		revealItemInDir(path);
	};

	const openToFinder = () => {
		if (type === "text") {
			return revealItemInDir(value);
		}

		const [file] = value;

		revealItemInDir(file);
	};

	const handleDelete = async () => {
		const db = await getDatabase();

		const matched = find(rootState.list, { id });

		if (!matched) return;

		let confirmed = true;

		if (clipboardStore.content.deleteConfirm) {
			confirmed = await deleteModal.confirm({
				centered: true,
				content: t("clipboard.hints.delete_modal_content"),
				afterClose() {
					// 关闭确认框后焦点还在，需要手动取消焦点
					(document.activeElement as HTMLElement)?.blur();
				},
			});
		}

		if (!confirmed) return;

		remove(rootState.list, { id });

		db.deleteFrom("history").where("id", "=", id).execute();
	};

	const handleContextMenu = async (event: MouseEvent) => {
		event.preventDefault();

		rootState.activeId = id;

		const items: ContextMenuItem[] = [
			{
				text: t("clipboard.button.context_menu.copy"),
				action: () => writeToClipboard(data),
			},
			{
				text: t("clipboard.button.context_menu.note"),
				action: handleNote,
			},
			{
				text: t("clipboard.button.context_menu.paste_as_plain_text"),
				hide: type !== "html" && type !== "rtf",
				action: pasteAsText,
			},
			{
				text: t("clipboard.button.context_menu.paste_as_path"),
				hide: type !== "files",
				action: pasteAsText,
			},
			{
				text: favorite
					? t("clipboard.button.context_menu.unfavorite")
					: t("clipboard.button.context_menu.favorite"),
				action: handleFavorite,
			},
			{
				text: t("clipboard.button.context_menu.open_in_browser"),
				hide: subtype !== "url",
				action: openToBrowser,
			},
			{
				text: t("clipboard.button.context_menu.send_email"),
				hide: subtype !== "email",
				action: () => openUrl(`mailto:${value}`),
			},
			{
				text: t("clipboard.button.context_menu.export_as_file"),
				hide: group !== "text",
				action: exportToFile,
			},
			{
				text: t("clipboard.button.context_menu.download_image"),
				hide: type !== "image",
				action: downloadImage,
			},
			{
				text: isMac
					? t("clipboard.button.context_menu.show_in_finder")
					: t("clipboard.button.context_menu.show_in_file_explorer"),
				hide: type !== "files" && subtype !== "path",
				action: openToFinder,
			},
			{
				text: t("clipboard.button.context_menu.delete"),
				action: handleDelete,
			},
		];

		const menu = await Menu.new();

		for await (const item of items.filter(({ hide }) => !hide)) {
			const menuItem = await MenuItem.new(item);

			await menu.append(menuItem);
		}

		menu.popup();
	};

	return {
		handleContextMenu,
		handleFavorite,
		handleDelete,
	};
};
