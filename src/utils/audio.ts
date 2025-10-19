import copyAudioURL from "@/assets/audio/copy.mp3";

let copyAudio: HTMLAudioElement | null = null;

export const playAudio = (url: string = copyAudioURL) => {
	copyAudio ??= new Audio(url);

	copyAudio.currentTime = 0;

	copyAudio.play();
};
