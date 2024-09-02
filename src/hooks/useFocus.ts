import { appWindow } from "@tauri-apps/api/window";

interface Props {
	onFocus?: () => void;
	onBlur?: () => void;
}

interface State {
	unlisten?: () => void;
}

export const useFocus = (props: Props) => {
	const { onFocus, onBlur } = props;

	const state = useReactive<State>({});

	const { run } = useDebounceFn(
		({ payload }) => {
			if (payload) {
				onFocus?.();
			} else {
				onBlur?.();
			}
		},
		{ wait: isMac() ? 0 : 100 },
	);

	useMount(async () => {
		state.unlisten = await appWindow.onFocusChanged(run);
	});

	useUnmount(() => {
		state.unlisten?.();
	});
};
