import ShortcutKey from "@/components/ShortcutKey";
import { Card, Flex, Switch } from "antd";
import { useSnapshot } from "valtio";
import DefaultFocus from "./components/DefaultFocus";
import DoubleClickFeedback from "./components/DoubleClickFeedback";
import HistoryCapacity from "./components/HistoryCapacity";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { wakeUpKey, enableAudio } = useSnapshot(clipboardStore);

	return (
		<Flex vertical gap="middle">
			<Card title="基础设置">
				<Flex vertical gap="large">
					<Flex align="center">
						<span>唤醒窗口：</span>
						<ShortcutKey
							defaultValue={wakeUpKey}
							onChange={(value) => {
								clipboardStore.wakeUpKey = value;
							}}
						/>
					</Flex>

					<Flex align="center">
						启用音效：
						<Switch
							checked={enableAudio}
							onChange={(value) => {
								clipboardStore.enableAudio = value;
							}}
						/>
					</Flex>

					<SearchPosition />

					<DefaultFocus />

					<DoubleClickFeedback />

					<WindowPosition />
				</Flex>
			</Card>

			<HistoryCapacity />
		</Flex>
	);
};

export default Clipboard;
