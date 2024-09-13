import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { open } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { join, sep } from "@tauri-apps/api/path";
import { Button, Flex, message } from "antd";
import { isString } from "antd/es/button";
import type { FC } from "react";
import type { State } from "../..";

const SavePath: FC<{ state: State }> = (props) => {
	const { state } = props;

	const handleClick = async () => {
		try {
			const select = await open({ directory: true });

			if (!isString(select)) return;

			state.spinning = true;

			const dirName = await moveData(getSaveDataDir(), select);

			if (!dirName) return;

			globalStore.env.saveDataDir = await join(select, dirName);

			state.spinning = false;

			emit(LISTEN_KEY.CHANGE_DATA_FILE);

			message.success("更改成功");
		} catch (error: any) {
			state.spinning = false;

			message.error(error);
		}
	};

	return (
		<ProList header="存储路径">
			<ProListItem
				title={
					<Flex vertical align="flex-start" gap={2}>
						自定义存储路径
						<span
							className="color-3 hover:color-primary cursor-pointer break-all text-12 transition"
							onMouseDown={() => {
								previewPath(getSaveDataDir());
							}}
						>
							{getSaveDataDir().replace(new RegExp(`${sep}$`, "g"), "")}
						</span>
					</Flex>
				}
			>
				<Button onClick={handleClick}>更改</Button>
			</ProListItem>
		</ProList>
	);
};

export default SavePath;
