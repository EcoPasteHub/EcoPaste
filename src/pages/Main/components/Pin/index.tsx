import UnoIcon from "@/components/UnoIcon";
import clsx from "clsx";
import { MainContext } from "../..";

const Pin = () => {
	const { state } = useContext(MainContext);

	useKeyPress(PRESET_SHORTCUT.FIXED_WINDOW, () => {
		togglePin();
	});

	const togglePin = () => {
		state.pin = !state.pin;
	};

	return (
		<UnoIcon
			hoverable
			active={state.pin}
			name="i-lets-icons:pin"
			className={clsx({ "-rotate-45": !state.pin })}
			onMouseDown={togglePin}
		/>
	);
};

export default Pin;
