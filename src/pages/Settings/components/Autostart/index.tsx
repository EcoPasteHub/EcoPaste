import { Switch } from "antd";
import { disable, enable, isEnabled } from "tauri-plugin-autostart-api";
import { useSnapshot } from "valtio";

const Autostart = () => {
	const { autostart } = useSnapshot(store);

	useMount(async () => {
		const enabled = await isEnabled();

		if (enabled === autostart) return;

		handleChange();
	});

	useUpdateEffect(() => {
		if (autostart) {
			enable();
		} else {
			disable();
		}
	}, [autostart]);

	const handleChange = () => {
		store.autostart = !store.autostart;
	};

	return <Switch checked={autostart} onChange={handleChange} />;
};

export default Autostart;
