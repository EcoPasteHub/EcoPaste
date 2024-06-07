import ShortcutKey from "@/components/ShortcutKey";
import { Card, Flex, Switch } from "antd";
import { useSnapshot } from "valtio";
import Record from "./components/Record";

const Clipboard = () => {
	const { wakeUpKey, enableSound } = useSnapshot(clipboardStore);

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
							checked={enableSound}
							onChange={(value) => {
								clipboardStore.enableSound = value;
							}}
						/>
					</Flex>
				</Flex>
			</Card>
			<Record />
		</Flex>
	);
};

export default Clipboard;
