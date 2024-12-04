import { listen } from "@tauri-apps/api/event";
import { noop } from "lodash-es";

export const useTauriListen = <T>(...args: Parameters<typeof listen<T>>) => {
	const state = useReactive({
		unlisten: noop,
	});

	useMount(async () => {
		state.unlisten = await listen<T>(...args);
	});

	useUnmount(state.unlisten);
};
