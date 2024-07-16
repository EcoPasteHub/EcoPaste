import Icon from "@/components/Icon";
import { appWindow } from "@tauri-apps/api/window";
import type { Timeout } from "ahooks/lib/useRequest/src/types";
import { Flex } from "antd";
import clsx from "clsx";
import Tab from "./components/Tab";

interface State {
	pin?: boolean;
	delay: number;
}

let timer: Timeout;

const Header = () => {
	const state = useReactive<State>({
		delay: 0,
	});

	useMount(async () => {
		if (isWin()) {
			state.delay = 100;
		}

		appWindow.onFocusChanged(async ({ payload }) => {
			clearTimeout(timer);

			/**
			 * 背景：在 Windows 系统上，拖动窗口会多次触发 `onFocusChanged` 事件，导致窗口被失焦关闭
			 * 解决方案：给 `onFocusChanged` 事件加个定时器，用最新的状态做变更
			 */
			timer = setTimeout(() => {
				if (payload || state.pin) return;

				hideWindow();
			}, state.delay);
		});
	});

	return (
		<Flex
			data-tauri-drag-region
			align="center"
			justify="space-between"
			gap="small"
			className="color-2 px-12 pb-12 text-18"
		>
			<Tab />

			<Icon
				hoverable
				active={state.pin}
				name="i-ri:pushpin-2-line"
				className={clsx("min-w-18", { "rotate-45": state.pin })}
				onMouseDown={() => {
					state.pin = !state.pin;
				}}
			/>
		</Flex>
	);
};

export default Header;
