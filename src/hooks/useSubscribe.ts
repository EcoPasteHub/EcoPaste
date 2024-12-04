import { noop } from "lodash-es";
import { subscribe } from "valtio";

export const useSubscribe = (...args: Parameters<typeof subscribe>) => {
	const state = useReactive({
		unsubscribe: noop,
	});

	useMount(async () => {
		state.unsubscribe = subscribe(...args);
	});

	useUnmount(state.unsubscribe);
};
