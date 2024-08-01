import Hotkey from "@/components/Hotkey";
import ProSwitch from "@/components/ProSwitch";
import { Flex, List } from "antd";
import { useSnapshot } from "valtio";
import ClickFeedback from "./components/ClickFeedback";
import HistoryRecord from "./components/HistoryRecord";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { wakeUpKey, copyAudio, searchDefaultFocus, singleClick, doubleClick } =
		useSnapshot(clipboardStore);

	return (
		<Flex data-tauri-drag-region vertical gap="middle">
			<List bordered header="窗口设置">
				<List.Item
					actions={[
						<Hotkey
							key={1}
							defaultValue={wakeUpKey}
							onChange={(value) => {
								clipboardStore.wakeUpKey = value;
							}}
						/>,
					]}
				>
					<List.Item.Meta title="唤醒窗口" />
				</List.Item>

				<WindowPosition />
			</List>

			<List bordered header="音效设置">
				<ProSwitch
					title="复制音效"
					value={copyAudio}
					onChange={(value) => {
						clipboardStore.copyAudio = value;
					}}
				/>
			</List>

			<List bordered header="搜索框设置">
				<SearchPosition key={1} />

				<ProSwitch
					title="默认聚焦"
					description="每次打开窗口时，自动聚焦搜索框"
					value={searchDefaultFocus}
					onChange={(value) => {
						clipboardStore.searchDefaultFocus = value;
					}}
				/>

				{/* <ProSwitch
					title="自动清除"
					description="每次打开窗口时，自动清除搜索框"
					value={searchDefaultFocus}
					onChange={(value) => {
						clipboardStore.searchDefaultFocus = value;
					}}
				/> */}
			</List>

			<List bordered header="点击反馈">
				<ClickFeedback
					label="单击反馈"
					value={singleClick}
					onChange={(value) => {
						clipboardStore.singleClick = value;
					}}
				/>

				<ClickFeedback
					label="双击反馈"
					value={doubleClick}
					onChange={(value) => {
						clipboardStore.doubleClick = value;
					}}
				/>
			</List>

			<HistoryRecord />
		</Flex>
	);
};

export default Clipboard;
