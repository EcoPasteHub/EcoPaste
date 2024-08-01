import Hotkey from "@/components/Hotkey";
import { emit } from "@tauri-apps/api/event";
import { Button, Flex, List, Popconfirm, Select, Switch } from "antd";
import { useSnapshot } from "valtio";
import DoubleClickFeedback from "./components/DoubleClickFeedback";
import SearchPosition from "./components/SearchPosition";
import WindowPosition from "./components/WindowPosition";

const Clipboard = () => {
	const { wakeUpKey, enableAudio } = useSnapshot(clipboardStore);

	return (
		<Flex data-tauri-drag-region vertical gap="middle">
			<List bordered header="窗口设置">
				<List.Item
					actions={[
						<Hotkey
							key={1}
							defaultValue={wakeUpKey}
							onChange={(value) => {
								clipboardStore.wakeUpKey = value;
							}}
						/>,
					]}
				>
					<List.Item.Meta title="显示/隐藏" />
				</List.Item>

				<List.Item actions={[<WindowPosition key={1} />]}>
					<List.Item.Meta title="窗口位置" />
				</List.Item>
			</List>

			<List bordered header="音效设置">
				<List.Item
					actions={[
						<Switch
							key={1}
							checked={enableAudio}
							onChange={(value) => {
								clipboardStore.enableAudio = value;
							}}
						/>,
					]}
				>
					<List.Item.Meta title="复制音效" />
				</List.Item>
			</List>

			<List bordered header="搜索框设置">
				<List.Item actions={[<SearchPosition key={1} />]}>
					<List.Item.Meta title="位置" />
				</List.Item>

				<List.Item actions={[<Switch key={1} />]}>
					<List.Item.Meta
						title="默认聚焦"
						description="每次打开窗口时，自动聚焦搜索框"
					/>
				</List.Item>

				<List.Item actions={[<Switch key={0} />]}>
					<List.Item.Meta
						title="自动清除"
						description="每次打开窗口时，自动清除搜索框"
					/>
				</List.Item>
			</List>

			<List bordered header="点击反馈">
				<List.Item actions={[<DoubleClickFeedback key={1} />]}>
					<List.Item.Meta title="单击反馈" />
				</List.Item>

				<List.Item actions={[<DoubleClickFeedback key={1} />]}>
					<List.Item.Meta title="双击反馈" />
				</List.Item>
			</List>

			<List
				bordered
				header="历史记录"
				footer={
					<Popconfirm
						title="确定要清除历史记录？"
						onConfirm={() => emit(LISTEN_KEY.CLEAR_HISTORY)}
					>
						<Button block danger>
							清除历史记录
						</Button>
					</Popconfirm>
				}
			>
				<List.Item actions={[<Select key={1} />]}>
					<List.Item.Meta title="记录容量" />
				</List.Item>
			</List>
		</Flex>
	);
};

export default Clipboard;
