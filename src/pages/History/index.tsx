import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { emit } from "@tauri-apps/api/event";
import { Button, InputNumber, Select, Space } from "antd";
import { useSnapshot } from "valtio";

const History = () => {
	const { history } = useSnapshot(clipboardStore);

	const unitOptions = [
		{
			label: "天",
			value: 1,
		},
		{
			label: "周",
			value: 7,
		},
		{
			label: "月",
			value: 30,
		},
		{
			label: "年",
			value: 365,
		},
	];

	const handleClear = async () => {
		const yes = await ask("你确定要清除所有历史记录（包括收藏）吗？", {
			title: "清除历史记录",
			okLabel: "确定",
			cancelLabel: "取消",
			type: "warning",
		});

		if (!yes) return;

		emit(LISTEN_KEY.CLEAR_HISTORY);
	};

	return (
		<ProList
			header="历史记录"
			footer={
				<Button block danger onClick={handleClear}>
					清除历史记录
				</Button>
			}
		>
			<ProListItem title="保留时长" description="输入 0 则表示永久保留">
				<Space.Compact>
					<InputNumber
						min={0}
						value={history.duration}
						onChange={(value) => {
							clipboardStore.history.duration = value ?? 0;
						}}
					/>
					<Select
						value={history.unit}
						options={unitOptions}
						onChange={(value) => {
							clipboardStore.history.unit = value ?? 0;
						}}
					/>
				</Space.Compact>
			</ProListItem>
		</ProList>
	);
};

export default History;
