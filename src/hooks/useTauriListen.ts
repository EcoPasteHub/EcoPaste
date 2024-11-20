import { listen } from "@tauri-apps/api/event";
import { noop } from "lodash-es";

export const useTauriListen = (...args: Parameters<typeof listen>) => {
	const state = useReactive({
		unlisten: noop,
	});

	useMount(async () => {
		state.unlisten = await listen(...args);
	});

	useUnmount(state.unlisten);
};
