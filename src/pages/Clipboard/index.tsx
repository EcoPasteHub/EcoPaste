import { Card, Flex, Switch } from "antd";
import Record from "./components/Record";

const Clipboard = () => {
	return (
		<Flex vertical gap="middle">
			<Card title="基础设置">
				<Flex align="center">
					启用音效：
					<Switch />
				</Flex>
			</Card>
			<Record />
		</Flex>
	);
};

export default Clipboard;
