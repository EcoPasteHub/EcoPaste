import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { debounce } from "lodash-es";

interface Props {
	onFocus?: () => void;
	onBlur?: () => void;
}

export const useTauriFocus = (props: Props) => {
	const { onFocus, onBlur } = props;

	useMount(async () => {
		const appWindow = getCurrentWebviewWindow();

		const wait = isMac() ? 0 : 100;

		const debounced = debounce(({ payload }) => {
			if (payload) {
				onFocus?.();
			} else {
				onBlur?.();
			}
		}, wait);

		appWindow.onFocusChanged(debounced);
	});
};
