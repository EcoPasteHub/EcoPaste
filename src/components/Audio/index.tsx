import copyAudio from "@/assets/audio/copy.mp3";
import Icon, { type IconProps } from "../Icon";

export interface AudioProps {
	src?: string;
	hiddenIcon?: boolean;
	iconProps?: IconProps;
}

export interface AudioRef {
	play: () => void;
}

const Audio = forwardRef<AudioRef, AudioProps>((props, ref) => {
	const { hiddenIcon, iconProps, src = copyAudio } = props;

	const audioRef = useRef<HTMLAudioElement>(null);

	useImperativeHandle(ref, () => ({
		play: playAudio,
	}));

	const playAudio = () => {
		audioRef.current?.play();
	};

	return (
		<>
			<Icon
				{...iconProps}
				hoverable
				hidden={hiddenIcon}
				name="i-iconamoon:volume-up-light"
				onClick={playAudio}
			/>

			<audio ref={audioRef} src={src === "copy" ? copyAudio : src} />
		</>
	);
});

export default Audio;
