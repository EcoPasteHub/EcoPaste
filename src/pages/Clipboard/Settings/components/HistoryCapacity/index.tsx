import { emit } from "@tauri-apps/api/event";
import { Button, Card, Popconfirm, Slider } from "antd";
import type { SliderBaseProps } from "antd/es/slider";
import { keys } from "lodash-es";
import { useSnapshot } from "valtio";

const useDuration = () => {
	const { t } = useTranslation();

	const labels = [
		t("preference.clipboard.history_capacity.label.duration.day"),
		t("preference.clipboard.history_capacity.label.duration.week"),
		t("preference.clipboard.history_capacity.label.duration.month"),
		t("preference.clipboard.history_capacity.label.duration.quarter"),
		t("preference.clipboard.history_capacity.label.duration.half_year"),
		t("preference.clipboard.history_capacity.label.duration.year"),
		t("preference.clipboard.history_capacity.label.duration.unlimited"),
	];
	const values = [1, 7, 30, 90, 180, 365, 0];
	const steps = labels.length;
	const step = Math.round(100 / (steps - 1));

	return {
		labels,
		values,
		steps,
		step,
	};
};

const HistoryCapacity = () => {
	const { historyCapacity } = useSnapshot(clipboardStore);
	const { t } = useTranslation();
	const { labels, values, steps, step } = useDuration();

	const marks = useCreation(() => {
		const marks: SliderBaseProps["marks"] = {};

		for (let i = 0; i < steps; i++) {
			marks[Math.min(i * step, 100)] = labels[i];
		}

		return marks;
	}, [labels]);

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
			title={t("preference.clipboard.history_capacity.title")}
			extra={
				<Popconfirm
					placement="bottomRight"
					title={t("preference.clipboard.history_capacity.hints.clear_confirm")}
					onConfirm={() => emit(LISTEN_KEY.CLEAR_HISTORY)}
				>
					<Button danger ghost type="primary">
						{t("preference.clipboard.history_capacity.button.clear")}
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
