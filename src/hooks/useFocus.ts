import { listen } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";

interface Props {
	onFocus?: () => void;
	onBlur?: () => void;
}

export const useFocus = (props: Props) => {
	const { onFocus, onBlur } = props;

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
		const appWindow = getCurrentWebviewWindow();

		appWindow.onFocusChanged(run);

		listen(LISTEN_KEY.MACOS_PANEL_FOCUS, run);
	});
};
