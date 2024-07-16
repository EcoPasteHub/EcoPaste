import { appWindow } from "@tauri-apps/api/window";
import type { Timeout } from "ahooks/lib/useRequest/src/types";

interface Props {
	onFocus?: () => void;
	onBlur?: () => void;
}

interface State {
	delay: number;
	timer?: Timeout;
	unlisten?: () => void;
}

export const useFocus = (props: Props) => {
	const { onFocus, onBlur } = props;

	const state = useReactive<State>({
		delay: 0,
	});

	useMount(async () => {
		if (isWin()) {
			state.delay = 100;
		}

		state.unlisten = await appWindow.onFocusChanged(({ payload }) => {
			clearTimeout(state.timer);

			/**
			 * 背景：在 Windows 系统上，拖动窗口会多次触发 `onFocusChanged` 事件，导致窗口被失焦关闭
			 * 解决方案：给 `onFocusChanged` 事件加个定时器，用最新的状态做变更
			 */
			state.timer = setTimeout(() => {
				if (payload) {
					onFocus?.();
				} else {
					onBlur?.();
				}
			}, state.delay);
		});
	});

	useUnmount(() => {
		state.unlisten?.();
	});
};
