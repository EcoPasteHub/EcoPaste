import { invoke } from "@tauri-apps/api";
import { Card, Flex, Segmented, Switch } from "antd";

const Settings = () => {
	return (
		<Flex vertical gap="middle">
			<Card title="通用设置">
				<Flex vertical gap="large">
					<Flex align="center">
						<span>开机自启：</span>
						<Switch />
					</Flex>
					{/* <Flex align="center">
						<span>唤醒窗口：</span>
					</Flex> */}
					<Flex align="center">
						<span>切换分组：</span>
						<Segmented
							defaultValue="click"
							options={[
								{
									label: "点击",
									value: "click",
								},
								{
									label: "悬浮",
									value: "hover",
								},
							]}
						/>
					</Flex>
					<Flex align="center">
						<span>主题模式：</span>
						<Segmented
							defaultValue="auto"
							options={[
								{
									label: "自动",
									value: "auto",
								},
								{
									label: "亮色",
									value: "light",
								},
								{
									label: "暗色",
									value: "dark",
								},
							]}
							onChange={(value) => {
								invoke("plugin:theme|set_theme", {
									theme: value,
								});
							}}
						/>
					</Flex>
				</Flex>
			</Card>
		</Flex>
	);
};

export default Settings;
