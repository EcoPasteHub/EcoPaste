import PlayAudio from "@/components/PlayAudio";
import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { Typography } from "antd";
import { useSnapshot } from "valtio";
import AutoPaste from "./components/AutoPaste";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { audio, search, content } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	return (
		<>
			<ProList header={t("preference.clipboard.window_settings.title")}>
				<WindowPosition />
			</ProList>

			<ProList header={t("preference.clipboard.audio_settings.title")}>
				<ProSwitch
					title={t("preference.clipboard.audio_settings.label.copy_audio")}
					value={audio.copy}
					onChange={(value) => {
						clipboardStore.audio.copy = value;
					}}
				>
					<PlayAudio
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
						<>
							{t("preference.clipboard.content_settings.hints.image_ocr")}{" "}
							<Typography.Link href="https://github.com/tesseract-ocr/tesseract">
								tesseract
							</Typography.Link>
						</>
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
					value={content.copyPlainText}
					onChange={(value) => {
						clipboardStore.content.copyPlainText = value;
					}}
				/>
			</ProList>
		</>
	);
};

export default Clipboard;
