import ShortcutKey from "@/components/ShortcutKey";
import { Card, Flex, Switch } from "antd";
import { useSnapshot } from "valtio";
import ThemeMode from "./components/ThemeMode";
import TrayClick from "./components/TrayClick";

const Settings = () => {
	const { autoStart, wakeUpKey, autoUpdate } = useSnapshot(globalStore);

	return (
		<Flex vertical gap="middle">
			<Card title="基础设置">
				<Flex vertical gap="large">
					<Flex align="center">
						<span>开机自启：</span>
						<Switch
							checked={autoStart}
							onChange={(value) => {
								globalStore.autoStart = value;
							}}
						/>
					</Flex>

					<Flex align="center">
						<span>自动更新：</span>
						<Switch
							checked={autoUpdate}
							onChange={(value) => {
								globalStore.autoUpdate = value;
							}}
						/>
					</Flex>

					<Flex align="center">
						<span>唤醒窗口：</span>
						<ShortcutKey
							defaultValue={wakeUpKey}
							onChange={(value) => {
								globalStore.wakeUpKey = value;
							}}
						/>
					</Flex>

					<ThemeMode />

					<TrayClick />
				</Flex>
			</Card>
		</Flex>
	);
};

export default Settings;
