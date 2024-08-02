import ProList from "@/components/ProList";
import ProSelect from "@/components/ProSelect";
import { emit } from "@tauri-apps/api/event";
import { Button } from "antd";
import { useSnapshot } from "valtio";

const HistoryRecord = () => {
	const { historyDuration } = useSnapshot(clipboardStore);

	const options = [
		{
			label: "永久",
			value: 0,
		},
		{
			label: "一天",
			value: 1,
		},
		{
			label: "一周",
			value: 7,
		},
		{
			label: "一月",
			value: 30,
		},
		{
			label: "半年",
			value: 180,
		},
		{
			label: "一年",
			value: 365,
		},
	];

	const handleClear = async () => {
		const yes = await ask("你确定要清除所有历史记录吗？", {
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
			<ProSelect
				title="保留时长"
				value={historyDuration}
				options={options}
				onChange={(value) => {
					clipboardStore.historyDuration = value;
				}}
			/>
		</ProList>
	);
};

export default HistoryRecord;
