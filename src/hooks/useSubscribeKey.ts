import { noop } from "lodash-es";
import { subscribeKey } from "valtio/utils";

export const useSubscribeKey: typeof subscribeKey = (...args) => {
	const state = useReactive({
		unsubscribe: noop,
	});

	useMount(async () => {
		state.unsubscribe = subscribeKey(...args);
	});

	useUnmount(state.unsubscribe);

	return state.unsubscribe;
};
