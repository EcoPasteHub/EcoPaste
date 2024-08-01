import ProSwitch from "@/components/ProSwitch";
import { Flex, List } from "antd";
import { useSnapshot } from "valtio";
import Language from "./components/Language";
import ThemeMode from "./components/ThemeMode";

const Settings = () => {
	const { autoStart, autoUpdate } = useSnapshot(globalStore);

	return (
		<Flex vertical gap="middle">
			<List bordered header="应用设置">
				<ProSwitch
					title="登录时启动"
					value={autoStart}
					onChange={(value) => {
						globalStore.autoStart = value;
					}}
				/>

				<ProSwitch
					title="自动检查更新"
					value={autoUpdate}
					onChange={(value) => {
						globalStore.autoUpdate = value;
					}}
				/>
			</List>

			<List bordered header="显示设置">
				<Language />

				<ThemeMode />
			</List>
		</Flex>
	);
};

export default Settings;
