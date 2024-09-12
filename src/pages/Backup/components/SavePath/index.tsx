import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { open } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { join } from "@tauri-apps/api/path";
import { Button, Flex } from "antd";
import { isString } from "antd/es/button";
import type { FC } from "react";
import { useSnapshot } from "valtio";
import type { State } from "../..";

const SavePath: FC<{ state: State }> = (props) => {
	const { state } = props;
	const { env } = useSnapshot(globalStore);

	const handleClick = async () => {
		const select = await open({ directory: true });

		if (!isString(select)) return;

		state.spinning = true;

		const dirName = await moveData(getSaveDataDir(), select);

		globalStore.env.saveDataDir = await join(select, dirName);

		state.spinning = false;

		emit(LISTEN_KEY.CHANGE_DATA_FILE);
	};

	return (
		<ProList header="存储路径">
			<ProListItem
				title={
					<Flex vertical gap={2}>
						自定义存储路径
						<span className="color-3 break-all text-12">{env.saveDataDir}</span>
					</Flex>
				}
			>
				<Button onClick={handleClick}>更改</Button>
			</ProListItem>
		</ProList>
	);
};

export default SavePath;
