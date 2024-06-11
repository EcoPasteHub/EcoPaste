import Icon from "@/components/Icon";
import { appWindow } from "@tauri-apps/api/window";
import { Flex } from "antd";
import clsx from "clsx";
import Search from "./components/Search";
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
			gap="small"
			className="color-2 pb-12 text-18"
		>
			<Flex align="center" gap="small" className="overflow-hidden">
				<Search />

				<Tab className="overflow-auto" />
			</Flex>

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
