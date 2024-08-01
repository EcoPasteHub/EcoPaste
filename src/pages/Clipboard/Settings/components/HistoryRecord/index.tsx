import ProList from "@/components/ProList";
import ProSelect from "@/components/ProSelect";
import { emit } from "@tauri-apps/api/event";
import { Button } from "antd";
import { useSnapshot } from "valtio";

const HistoryRecord = () => {
	const { historyDuration } = useSnapshot(clipboardStore);

	const [value, setValue] = useState(historyDuration);

	const options = useCreation(() => {
		return [
			{
				label: "永久",
				value: 0,
			},
			{
				label: `${value}天`,
				value: value,
			},
			{
				label: `${value}周`,
				value: value * 7,
			},
			{
				label: `${value}月`,
				value: value * 30,
			},
			{
				label: `${value}年`,
				value: value * 365,
			},
		];
	}, [value]);

	const handleChange = (value: number) => {
		setValue(value);

		clipboardStore.historyDuration = value;
	};

	const handleSearch = (value: string) => {
		let number = Number.parseInt(value);

		if (Number.isNaN(number) || number <= 0) {
			number = 1;
		}

		setValue(number);
	};

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
				showSearch
				title="保留时长"
				value={value}
				options={options}
				filterOption={false}
				onChange={handleChange}
				onSearch={handleSearch}
			/>
		</ProList>
	);
};

export default HistoryRecord;
