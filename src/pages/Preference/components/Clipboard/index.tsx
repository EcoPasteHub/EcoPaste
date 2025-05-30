import Audio from "@/components/Audio";
import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { Typography } from "antd";
import { useSnapshot } from "valtio";
import AutoPaste from "./components/AutoPaste";
import OperationButton from "./components/OperationButton";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const ClipboardSettings = () => {
	const { window, audio, search, content } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	return (
		<>
			<ProList header={t("preference.clipboard.window_settings.title")}>
				<WindowPosition />

				<ProSwitch
					title={t("preference.clipboard.window_settings.label.back_top")}
					description={t("preference.clipboard.window_settings.hints.back_top")}
					value={window.backTop}
					onChange={(value) => {
						clipboardStore.window.backTop = value;
					}}
				/>

				<ProSwitch
					title={t("preference.clipboard.window_settings.label.show_all")}
					value={window.showAll}
					onChange={(value) => {
						clipboardStore.window.showAll = value;
					}}
				/>
			</ProList>

			<ProList header={t("preference.clipboard.audio_settings.title")}>
				<ProSwitch
					title={t("preference.clipboard.audio_settings.label.copy_audio")}
					value={audio.copy}
					onChange={(value) => {
						clipboardStore.audio.copy = value;
					}}
				>
					<Audio
						iconProps={{
							size: 22,
							className: "flex!",
						}}
					/>
				</ProSwitch>
			</ProList>

			<ProList header={t("preference.clipboard.search_box_settings.title")}>
				<SearchPosition key={1} />

				<ProSwitch
					title={t(
						"preference.clipboard.search_box_settings.label.default_focus",
					)}
					description={t(
						"preference.clipboard.search_box_settings.hints.default_focus",
					)}
					value={search.defaultFocus}
					onChange={(value) => {
						clipboardStore.search.defaultFocus = value;
					}}
				/>

				<ProSwitch
					title={t("preference.clipboard.search_box_settings.label.auto_clear")}
					description={t(
						"preference.clipboard.search_box_settings.hints.auto_clear",
					)}
					value={search.autoClear}
					onChange={(value) => {
						clipboardStore.search.autoClear = value;
					}}
				/>
			</ProList>

			<ProList header={t("preference.clipboard.content_settings.title")}>
				<AutoPaste />

				<ProSwitch
					title={t("preference.clipboard.content_settings.label.image_ocr")}
					description={
						isLinux && (
							<>
								{t("preference.clipboard.content_settings.hints.image_ocr")}{" "}
								<Typography.Link href="https://github.com/tesseract-ocr/tesseract">
									tesseract
								</Typography.Link>
							</>
						)
					}
					value={content.ocr}
					onChange={(value) => {
						clipboardStore.content.ocr = value;
					}}
				/>

				<ProSwitch
					title={t("preference.clipboard.content_settings.label.copy_as_plain")}
					description={t(
						"preference.clipboard.content_settings.hints.copy_as_plain",
					)}
					value={content.copyPlain}
					onChange={(value) => {
						clipboardStore.content.copyPlain = value;
					}}
				/>

				<ProSwitch
					title={t(
						"preference.clipboard.content_settings.label.paste_as_plain",
					)}
					description={t(
						"preference.clipboard.content_settings.hints.paste_as_plain",
					)}
					value={content.pastePlain}
					onChange={(value) => {
						clipboardStore.content.pastePlain = value;
					}}
				/>

				<OperationButton />

				<ProSwitch
					title={t("preference.clipboard.content_settings.label.auto_favorite")}
					description={t(
						"preference.clipboard.content_settings.hints.auto_favorite",
					)}
					value={content.autoFavorite}
					onChange={(value) => {
						clipboardStore.content.autoFavorite = value;
					}}
				/>

				<ProSwitch
					title={t(
						"preference.clipboard.content_settings.label.delete_confirm",
					)}
					description={t(
						"preference.clipboard.content_settings.hints.delete_confirm",
					)}
					value={content.deleteConfirm}
					onChange={(value) => {
						clipboardStore.content.deleteConfirm = value;
					}}
				/>

				<ProSwitch
					title={t("preference.clipboard.content_settings.label.auto_sort")}
					description={t(
						"preference.clipboard.content_settings.hints.auto_sort",
					)}
					value={content.autoSort}
					onChange={(value) => {
						clipboardStore.content.autoSort = value;
					}}
				/>

				<ProSwitch
					title={t(
						"preference.clipboard.content_settings.label.show_original_content",
					)}
					description={t(
						"preference.clipboard.content_settings.hints.show_original_content",
					)}
					value={content.showOriginalContent}
					onChange={(value) => {
						clipboardStore.content.showOriginalContent = value;
					}}
				/>
			</ProList>
		</>
	);
};

export default ClipboardSettings;
