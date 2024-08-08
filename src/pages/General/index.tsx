import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useSnapshot } from "valtio";
import Language from "./components/Language";
import ThemeMode from "./components/ThemeMode";

const General = () => {
	const { app } = useSnapshot(globalStore);

	return (
		<>
			<ProList header="应用设置">
				<ProSwitch
					title="登录时启动"
					value={app.autoStart}
					onChange={(value) => {
						globalStore.app.autoStart = value;
					}}
				/>

				<ProSwitch
					title="自动检查更新"
					value={app.autoUpdate}
					onChange={(value) => {
						globalStore.app.autoUpdate = value;
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

export default General;
