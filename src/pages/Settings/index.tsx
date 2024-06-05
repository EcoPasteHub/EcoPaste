import ShortcutKey from "@/layouts/ShortcutKey";
import type { TabTrigger } from "@/types/store";
import { Card, Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";
import AutoStart from "./components/AutoStart";
import ThemeMode from "./components/ThemeMode";

interface Option {
	label: string;
	value: TabTrigger;
}

const Settings = () => {
	const { tabTrigger } = useSnapshot(store);

	const options: Option[] = [
		{
			label: "点击",
			value: "click",
		},
		{
			label: "悬浮",
			value: "hover",
		},
	];

	return (
		<Flex vertical gap="middle">
			<Card title="通用设置">
				<Flex vertical gap="large">
					<AutoStart />

					<Flex align="center">
						<span>唤醒窗口：</span>
						<ShortcutKey />
					</Flex>

					<Flex align="center">
						<span>切换分组：</span>
						<Segmented
							value={tabTrigger}
							options={options}
							onChange={(value) => {
								store.tabTrigger = value;
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
