import clsx from "clsx";
import UnoIcon from "@/components/UnoIcon";
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
      active={rootState.pinned}
      className={clsx({ "-rotate-45": !rootState.pinned })}
      hoverable
      name="i-lets-icons:pin"
      onMouseDown={togglePin}
    />
  );
};

export default WindowPin;
