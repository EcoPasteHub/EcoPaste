import Icon from "@/components/Icon";
import { Flex } from "antd";
import clsx from "clsx";
import Tab from "./components/Tab";

interface State {
	pin?: boolean;
}

const Header = () => {
	const state = useReactive<State>({});

	useFocus({
		onBlur() {
			if (state.pin) return;

			hideWindow();
		},
	});

	return (
		<Flex
			data-tauri-drag-region
			align="center"
			justify="space-between"
			gap="small"
			className="color-2 px-12 text-18"
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
