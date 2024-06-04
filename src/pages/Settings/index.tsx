import { appWindow } from "@tauri-apps/api/window";
import { Card, Flex, Segmented } from "antd";
import Autostart from "./components/Autostart";
import ThemeMode from "./components/ThemeMode";

const Settings = () => {
	useMount(() => {
		appWindow.onThemeChanged(({ payload }) => {
			if (store.theme !== "auto") return;

			store.isDark = payload === "dark";
		});
	});

	return (
		<Flex vertical gap="middle">
			<Card title="通用设置">
				<Flex vertical gap="large">
					<Flex align="center">
						<span>开机自启：</span>
						<Autostart />
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
						<ThemeMode />
					</Flex>
				</Flex>
			</Card>
		</Flex>
	);
};

export default Settings;
