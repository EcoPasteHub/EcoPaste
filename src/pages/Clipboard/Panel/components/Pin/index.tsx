import Icon from "@/components/Icon";
import clsx from "clsx";
import { ClipboardPanelContext } from "../..";

const Pin = () => {
	const { state } = useContext(ClipboardPanelContext);

	return (
		<Icon
			hoverable
			active={state.pin}
			name="i-lets-icons:pin"
			className={clsx({ "-rotate-45": !state.pin })}
			onMouseDown={() => {
				state.pin = !state.pin;
			}}
		/>
	);
};

export default Pin;
