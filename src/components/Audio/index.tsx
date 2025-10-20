import copyAudio from "@/assets/audio/copy.mp3";

export interface AudioRef {
  play: () => void;
}

interface AudioProps {
  src?: string;
}

const Audio = forwardRef<AudioRef, AudioProps>((props, ref) => {
  const { src = copyAudio } = props;
  const audioRef = useRef<HTMLAudioElement>(null);

  useImperativeHandle(ref, () => ({
    play: playAudio,
  }));

  const playAudio = () => {
    audioRef.current?.play();
  };

  return <audio ref={audioRef} src={src} />;
});

export default Audio;
