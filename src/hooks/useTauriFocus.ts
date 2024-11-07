import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";

interface Props {
	onFocus?: () => void;
	onBlur?: () => void;
}

export const useTauriFocus = (props: Props) => {
	const { onFocus, onBlur } = props;

	const { run } = useDebounceFn(
		({ payload }) => {
			if (payload) {
				onFocus?.();
			} else {
				onBlur?.();
			}
		},
		{
			wait: isMac() ? 0 : 100,
		},
	);

	useMount(async () => {
		const appWindow = getCurrentWebviewWindow();

		appWindow.onFocusChanged(run);
	});
};
