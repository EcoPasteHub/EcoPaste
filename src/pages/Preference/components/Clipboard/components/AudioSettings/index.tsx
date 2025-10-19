import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import UnoIcon from "@/components/UnoIcon";
import { useSnapshot } from "valtio";

const AudioSettings = () => {
	const { audio } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	return (
		<ProList header={t("preference.clipboard.audio_settings.title")}>
			<ProSwitch
				title={t("preference.clipboard.audio_settings.label.copy_audio")}
				value={audio.copy}
				onChange={(value) => {
					clipboardStore.audio.copy = value;
				}}
			>
				<UnoIcon
					hoverable
					size={22}
					name="i-iconamoon:volume-up-light"
					className="flex!"
					onClick={() => playAudio()}
				/>
			</ProSwitch>
		</ProList>
	);
};

export default AudioSettings;
