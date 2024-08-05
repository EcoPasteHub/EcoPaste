import PlayAudio from "@/components/PlayAudio";
import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useSnapshot } from "valtio";
import ClickFeedback from "./components/ClickFeedback";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const {
		copyAudio,
		searchDefaultFocus,
		singleClick,
		doubleClick,
		searchAutoClear,
	} = useSnapshot(clipboardStore);

	return (
		<>
			<ProList header="窗口设置">
				<WindowPosition />
			</ProList>

			<ProList header="音效设置">
				<ProSwitch
					title="复制音效"
					value={copyAudio}
					onChange={(value) => {
						clipboardStore.copyAudio = value;
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
					value={searchDefaultFocus}
					onChange={(value) => {
						clipboardStore.searchDefaultFocus = value;
					}}
				/>

				<ProSwitch
					title="自动清除"
					description="每次打开窗口时，自动清除搜索框内容"
					value={searchAutoClear}
					onChange={(value) => {
						clipboardStore.searchAutoClear = value;
					}}
				/>
			</ProList>

			<ProList header="点击反馈">
				<ClickFeedback
					title="单击反馈"
					value={singleClick}
					onChange={(value) => {
						clipboardStore.singleClick = value;
					}}
				/>

				<ClickFeedback
					title="双击反馈"
					value={doubleClick}
					onChange={(value) => {
						clipboardStore.doubleClick = value;
					}}
				/>
			</ProList>
		</>
	);
};

export default Clipboard;
