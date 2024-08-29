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

			<Flex align="center" gap={4} className="color-2 text-18">
				<Icon
					hoverable
					active={state.pin}
					name="i-lets-icons:pin"
					className={clsx({ "-rotate-45": !state.pin })}
					onMouseDown={() => {
						state.pin = !state.pin;
					}}
				/>

				<Icon
					hoverable
					name="i-lets-icons:setting-alt-line"
					onClick={() => {
						showWindow("preference");
					}}
				/>
			</Flex>
		</Flex>
	);
};

export default Header;
