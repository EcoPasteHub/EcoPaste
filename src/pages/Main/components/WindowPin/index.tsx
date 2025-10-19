import UnoIcon from "@/components/UnoIcon";
import clsx from "clsx";
import { MainContext } from "../..";

const WindowPin = () => {
	const { rootState } = useContext(MainContext);

	useKeyPress(PRESET_SHORTCUT.FIXED_WINDOW, () => {
		togglePin();
	});

	useTauriFocus({
		onBlur() {
			if (rootState.pinned) return;

			hideWindow();
		},
	});

	const togglePin = () => {
		rootState.pinned = !rootState.pinned;
	};

	return (
		<UnoIcon
			hoverable
			active={rootState.pinned}
			name="i-lets-icons:pin"
			className={clsx({ "-rotate-45": !rootState.pinned })}
			onMouseDown={togglePin}
		/>
	);
};

export default WindowPin;
