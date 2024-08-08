import PlayAudio from "@/components/PlayAudio";
import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useSnapshot } from "valtio";
import AutoStart from "./components/AutoPaste";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { audio, search } = useSnapshot(clipboardStore);

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
				<AutoStart />
			</ProList>
		</>
	);
};

export default Clipboard;
