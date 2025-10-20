import { useSnapshot } from "valtio";
import Audio, { type AudioRef } from "@/components/Audio";
import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import UnoIcon from "@/components/UnoIcon";

const AudioSettings = () => {
  const { audio } = useSnapshot(clipboardStore);
  const { t } = useTranslation();
  const audioRef = useRef<AudioRef>(null);

  return (
    <ProList header={t("preference.clipboard.audio_settings.title")}>
      <Audio ref={audioRef} />

      <ProSwitch
        onChange={(value) => {
          clipboardStore.audio.copy = value;
        }}
        title={t("preference.clipboard.audio_settings.label.copy_audio")}
        value={audio.copy}
      >
        <UnoIcon
          className="flex!"
          hoverable
          name="i-iconamoon:volume-up-light"
          onClick={() => {
            audioRef.current?.play();
          }}
          size={22}
        />
      </ProSwitch>
    </ProList>
  );
};

export default AudioSettings;
