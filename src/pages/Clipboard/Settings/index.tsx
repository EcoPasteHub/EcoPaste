import PlayAudio from "@/components/PlayAudio";
import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useSnapshot } from "valtio";
import AutoPaste from "./components/AutoPaste";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { audio, search, content } = useSnapshot(clipboardStore);

	return (
		<>
			<ProList header="窗口设置">
				<WindowPosition />
			</ProList>

			<ProList header="音效设置">
				<ProSwitch
					title="复制音效"
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

			<ProList header="搜索框设置">
				<SearchPosition key={1} />

				<ProSwitch
					title="默认聚焦"
					description="每次打开窗口时，自动聚焦搜索框"
					value={search.defaultFocus}
					onChange={(value) => {
						clipboardStore.search.defaultFocus = value;
					}}
				/>

				<ProSwitch
					title="自动清除"
					description="每次打开窗口时，自动清除搜索框内容"
					value={search.autoClear}
					onChange={(value) => {
						clipboardStore.search.autoClear = value;
					}}
				/>
			</ProList>

			<ProList header="剪切板内容">
				<AutoPaste />

				<ProSwitch
					title="图片OCR"
					description="可以暂时避免由于系统 OCR 支持不足导致应用崩溃的问题"
					value={content.ocr}
					onChange={(value) => {
						clipboardStore.content.ocr = value;
					}}
				/>

				<ProSwitch
					title="复制为纯文本"
					description="富文本和 HTML 格式在复制时仅保留纯文本内容"
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
