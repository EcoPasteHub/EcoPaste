import { emit } from "@tauri-apps/api/event";
import { Button, Card, Popconfirm, Slider } from "antd";
import type { SliderBaseProps } from "antd/es/slider";
import { keys } from "lodash-es";
import { useSnapshot } from "valtio";

const labels = ["天", "周", "月", "季度", "半年", "年", "无限制"];
const values = [1, 7, 30, 90, 180, 365, 0];
const steps = labels.length;
const step = Math.round(100 / (steps - 1));

const HistoryCapacity = () => {
	const { historyCapacity } = useSnapshot(clipboardStore);

	const marks = useCreation(() => {
		const marks: SliderBaseProps["marks"] = {};

		for (let i = 0; i < steps; i++) {
			marks[Math.min(i * step, 100)] = labels[i];
		}

		return marks;
	}, []);

	const value = useCreation(() => {
		const index = values.findIndex((item) => item === historyCapacity);

		return Number(keys(marks)[index]);
	}, [historyCapacity]);

	const handleChange = (value: number) => {
		const index = labels.findIndex((item) => item === marks[value]);

		clipboardStore.historyCapacity = values[index];
	};

	return (
		<Card
			title="历史记录容量"
			extra={
				<Popconfirm
					placement="bottomRight"
					title="确定要清空历史记录吗？"
					onConfirm={() => emit(LISTEN_KEY.CLEAR_HISTORY)}
				>
					<Button danger ghost type="primary">
						清空
					</Button>
				</Popconfirm>
			}
		>
			<Slider
				value={value}
				step={step}
				marks={marks}
				tooltip={{ formatter: null }}
				onChange={handleChange}
			/>
		</Card>
	);
};

export default HistoryCapacity;
