import { noop } from "lodash-es";
import { subscribe } from "valtio";

export const useImmediate = (...args: Parameters<typeof subscribe>) => {
	const state = useReactive({
		unsubscribe: noop,
	});

	useMount(async () => {
		const [, callback] = args;

		callback([]);

		state.unsubscribe = subscribe(...args);
	});

	useUnmount(state.unsubscribe);
};
