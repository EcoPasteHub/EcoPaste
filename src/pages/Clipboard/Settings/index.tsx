import Hotkey from "@/components/Hotkey";
import { useAutoAnimate } from "@formkit/auto-animate/react";
import { Card, Flex, Switch } from "antd";
import { useSnapshot } from "valtio";
import DefaultFocus from "./components/DefaultFocus";
import DoubleClickFeedback from "./components/DoubleClickFeedback";
import HistoryCapacity from "./components/HistoryCapacity";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { wakeUpKey, enableAudio, clickPaste } = useSnapshot(clipboardStore);
	const { t } = useTranslation();
	const [animationParent] = useAutoAnimate();

	return (
		<Flex vertical gap="middle">
			<Card title={t("preference.clipboard.basic.title")}>
				<Flex ref={animationParent} vertical gap="large">
					<Flex align="center">
						<span>{t("preference.clipboard.basic.label.wake_up_key")}：</span>
						<Hotkey
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

					<Flex align="center">
						{t("preference.clipboard.basic.label.click_paste")}：
						<Switch
							checked={clickPaste}
							onChange={(value) => {
								clipboardStore.clickPaste = value;
							}}
						/>
					</Flex>

					{!clickPaste && <DoubleClickFeedback />}

					<WindowPosition />
				</Flex>
			</Card>

			<HistoryCapacity />
		</Flex>
	);
};

export default Clipboard;
