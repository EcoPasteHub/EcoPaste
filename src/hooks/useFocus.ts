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
		{
			wait: isMac() ? 0 : 100,
		},
	);

	useMount(async () => {
		/**
		 * 背景：在 Windows 系统上，拖动窗口会多次触发 `onFocusChanged` 事件，导致窗口被失焦关闭
		 * 解决方案：给 `onFocusChanged` 事件加个防抖，用最新的状态做变更
		 */
		state.unlisten = await appWindow.onFocusChanged(run);
	});

	useUnmount(() => {
		state.unlisten?.();
	});
};
