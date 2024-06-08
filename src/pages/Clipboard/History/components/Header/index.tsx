import Icon from "@/components/Icon";
import { appWindow } from "@tauri-apps/api/window";
import { Flex } from "antd";
import clsx from "clsx";
import Tab from "./components/Tab";

interface State {
	pin?: boolean;
}

const Header = () => {
	const state = useReactive<State>({});

	useMount(() => {
		appWindow.onFocusChanged(({ payload }) => {
			if (payload || state.pin) return;

			hideWindow();
		});
	});

	return (
		<Flex
			data-tauri-drag-region
			align="center"
			justify="space-between"
			className="color-2 pb-12 text-18"
		>
			<Flex align="center" gap="small">
				<Icon hoverable name="i-lucide:search" />

				<Tab />
			</Flex>

			<Icon
				hoverable
				active={state.pin}
				name="i-ri:pushpin-2-line"
				className={clsx({ "rotate-45": state.pin })}
				onMouseDown={() => {
					state.pin = !state.pin;
				}}
			/>
		</Flex>
	);
};

export default Header;
