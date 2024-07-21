import ShortcutKey from "@/components/ShortcutKey";
import { Card, Flex, Switch } from "antd";
import { useSnapshot } from "valtio";
import DefaultFocus from "./components/DefaultFocus";
import DoubleClickFeedback from "./components/DoubleClickFeedback";
import HistoryCapacity from "./components/HistoryCapacity";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { wakeUpKey, enableAudio } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	return (
		<Flex vertical gap="middle">
			<Card title={t("preference.clipboard.basic.title")}>
				<Flex vertical gap="large">
					<Flex align="center">
						<span>{t("preference.clipboard.basic.label.wake_up_key")}：</span>
						<ShortcutKey
							defaultValue={wakeUpKey}
							onChange={(value) => {
								clipboardStore.wakeUpKey = value;
							}}
						/>
					</Flex>

					<Flex align="center">
						{t("preference.clipboard.basic.label.enable_audio")}：
						<Switch
							checked={enableAudio}
							onChange={(value) => {
								clipboardStore.enableAudio = value;
							}}
						/>
					</Flex>

					<SearchPosition />

					<DefaultFocus />

					<DoubleClickFeedback />

					<WindowPosition />
				</Flex>
			</Card>

			<HistoryCapacity />
		</Flex>
	);
};

export default Clipboard;
