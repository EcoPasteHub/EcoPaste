import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useSnapshot } from "valtio";
import Language from "./components/Language";
import ThemeMode from "./components/ThemeMode";

const Settings = () => {
	const { autoStart, autoUpdate } = useSnapshot(globalStore);

	return (
		<>
			<ProList header="应用设置">
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
			</ProList>

			<ProList header="外观设置">
				<Language />

				<ThemeMode />
			</ProList>
		</>
	);
};

export default Settings;
