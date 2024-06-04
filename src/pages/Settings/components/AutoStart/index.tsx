import { Flex, Switch } from "antd";
import { disable, enable, isEnabled } from "tauri-plugin-autostart-api";
import { useSnapshot } from "valtio";

const AutoStart = () => {
	const { autoStart } = useSnapshot(store);

	useMount(async () => {
		store.autoStart = await isEnabled();
	});

	useUpdateEffect(() => {
		if (autoStart) {
			enable();
		} else {
			disable();
		}
	}, [autoStart]);

	return (
		<Flex align="center">
			<span>开机自启：</span>
			<Switch
				checked={autoStart}
				onChange={(value) => {
					store.autoStart = value;
				}}
			/>
		</Flex>
	);
};

export default AutoStart;
