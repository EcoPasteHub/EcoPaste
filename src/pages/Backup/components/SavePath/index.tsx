import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { open } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { join } from "@tauri-apps/api/path";
import { Button, Flex, Spin } from "antd";
import { isString } from "antd/es/button";
import { useSnapshot } from "valtio";

const SavePath = () => {
	const { env } = useSnapshot(globalStore);
	const [spinning, { setTrue, setFalse }] = useBoolean();

	const handleClick = async () => {
		const select = await open({ directory: true });

		if (!isString(select)) return;

		setTrue();

		const dirName = await moveData(getSaveDataDir(), select);

		globalStore.env.saveDataDir = await join(select, dirName);

		setFalse();

		emit(LISTEN_KEY.CHANGE_DATA);
	};

	return (
		<>
			<ProList header="存储路径">
				<ProListItem
					title={
						<Flex vertical gap={2}>
							自定义存储路径
							<span className="color-3 break-all text-12">
								{env.saveDataDir}
							</span>
						</Flex>
					}
				>
					<Button onClick={handleClick}>更改</Button>
				</ProListItem>
			</ProList>

			<Spin fullscreen spinning={spinning} percent="auto" />
		</>
	);
};

export default SavePath;
