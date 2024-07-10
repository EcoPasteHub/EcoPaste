import ShortcutKey from "@/components/ShortcutKey";
import { Card, Flex, Switch } from "antd";
import { useSnapshot } from "valtio";
import AutoStart from "./components/AutoStart";
import ThemeMode from "./components/ThemeMode";

const Settings = () => {
	const { wakeUpKey, autoUpdate } = useSnapshot(globalStore);

	return (
		<Flex vertical gap="middle">
			<Card title="基础设置">
				<Flex vertical gap="large">
					<AutoStart />

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
				</Flex>
			</Card>
		</Flex>
	);
};

export default Settings;
