import { Flex, Switch } from "antd";
import { disable, enable, isEnabled } from "tauri-plugin-autostart-api";
import { useSnapshot } from "valtio";

const AutoStart = () => {
	const { autoStart } = useSnapshot(globalStore);

	useMount(async () => {
		globalStore.autoStart = await isEnabled();
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
					globalStore.autoStart = value;
				}}
			/>
		</Flex>
	);
};

export default AutoStart;
