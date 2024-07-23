import Hotkey from "@/components/Hotkey";
import { Card, Flex, Switch } from "antd";
import { useSnapshot } from "valtio";
import Language from "./components/Language";
import ThemeMode from "./components/ThemeMode";
import TrayClick from "./components/TrayClick";

const Settings = () => {
	const { autoStart, wakeUpKey, autoUpdate } = useSnapshot(globalStore);
	const { t } = useTranslation();

	return (
		<Flex vertical gap="middle">
			<Card title={t("preference.settings.basic.title")}>
				<Flex vertical gap="large">
					<Flex align="center">
						<span>{t("preference.settings.basic.label.auto_start")}：</span>
						<Switch
							checked={autoStart}
							onChange={(value) => {
								globalStore.autoStart = value;
							}}
						/>
					</Flex>

					<Flex align="center">
						<span>{t("preference.settings.basic.label.auto_update")}：</span>
						<Switch
							checked={autoUpdate}
							onChange={(value) => {
								globalStore.autoUpdate = value;
							}}
						/>
					</Flex>

					<Flex align="center">
						<span>{t("preference.settings.basic.label.wake_up_key")}：</span>
						<Hotkey
							defaultValue={wakeUpKey}
							onChange={(value) => {
								globalStore.wakeUpKey = value;
							}}
						/>
					</Flex>

					<ThemeMode />

					<Language />

					<TrayClick />
				</Flex>
			</Card>
		</Flex>
	);
};

export default Settings;
