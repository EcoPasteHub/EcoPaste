import Icon from "@/components/Icon";
import { Flex } from "antd";
import clsx from "clsx";
import { HistoryContext } from "../..";
import Tab from "./components/Tab";

const Header = () => {
	const { state } = useContext(HistoryContext);

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
			className="px-9"
		>
			<Tab />

			<Icon
				hoverable
				active={state.pin}
				name="i-ri:pushpin-2-line"
				className={clsx("color-2 min-w-18 text-18", { "rotate-45": state.pin })}
				onMouseDown={() => {
					state.pin = !state.pin;
				}}
			/>
		</Flex>
	);
};

export default Header;
