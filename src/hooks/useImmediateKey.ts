import { noop } from "lodash-es";
import { subscribeKey } from "valtio/utils";

export const useImmediateKey: typeof subscribeKey = (...args) => {
	const state = useReactive({
		unsubscribe: noop,
	});

	useMount(async () => {
		const [object, key, callback] = args;

		callback(object[key]);

		state.unsubscribe = subscribeKey(...args);
	});

	useUnmount(state.unsubscribe);

	return state.unsubscribe;
};
