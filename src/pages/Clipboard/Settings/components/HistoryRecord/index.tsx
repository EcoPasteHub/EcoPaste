import { emit } from "@tauri-apps/api/event";
import { Button, List, Popconfirm, Select } from "antd";
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

	return (
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
			<List.Item
				actions={[
					<Select
						key={1}
						showSearch
						value={value}
						options={options}
						filterOption={false}
						onChange={handleChange}
						onSearch={handleSearch}
					/>,
				]}
			>
				<List.Item.Meta title="保留时长" />
			</List.Item>
		</List>
	);
};

export default HistoryRecord;
