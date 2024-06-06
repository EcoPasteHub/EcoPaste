import ShortcutKey from "@/components/ShortcutKey";
import { Card, Flex, Switch } from "antd";
import Record from "./components/Record";

const Clipboard = () => {
	return (
		<Flex vertical gap="middle">
			<Card title="基础设置">
				<Flex vertical gap="large">
					<Flex align="center">
						<span>唤醒窗口：</span>
						<ShortcutKey />
					</Flex>

					<Flex align="center">
						启用音效：
						<Switch />
					</Flex>
				</Flex>
			</Card>
			<Record />
		</Flex>
	);
};

export default Clipboard;
