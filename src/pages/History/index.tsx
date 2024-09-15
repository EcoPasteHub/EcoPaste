import EcoSelect from "@/components/EcoSelect";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { InputNumber } from "antd";
import { useSnapshot } from "valtio";
import Delete from "./components/Delete";

const History = () => {
	const { history } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const unitOptions = [
		{
			label: t("preference.history.history.label.duration_unit.day"),
			value: 1,
		},
		{
			label: t("preference.history.history.label.duration_unit.week"),
			value: 7,
		},
		{
			label: t("preference.history.history.label.duration_unit.month"),
			value: 30,
		},
		{
			label: t("preference.history.history.label.duration_unit.year"),
			value: 365,
		},
	];

	return (
		<ProList header={t("preference.history.history.title")} footer={<Delete />}>
			<ProListItem
				title={t("preference.history.history.label.duration")}
				description={"输入 0 则表示永久保留，自动删除时会保留收藏"}
			>
				<InputNumber
					min={0}
					className="w-130"
					value={history.duration}
					addonAfter={
						<EcoSelect
							value={history.unit}
							options={unitOptions}
							onChange={(value) => {
								clipboardStore.history.unit = value ?? 0;
							}}
						/>
					}
					onChange={(value) => {
						clipboardStore.history.duration = value ?? 0;
					}}
				/>
			</ProListItem>
		</ProList>
	);
};

export default History;
