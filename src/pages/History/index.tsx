import EcoSelect from "@/components/EcoSelect";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { emit } from "@tauri-apps/api/event";
import { Button, InputNumber, Space } from "antd";
import { useSnapshot } from "valtio";

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

	const handleClear = async () => {
		const yes = await ask(t("preference.history.history.hints.clear_confirm"), {
			title: t("preference.history.history.label.clear_confirm_title"),
			okLabel: t("preference.history.history.button.clear_confirm"),
			cancelLabel: t("preference.history.history.button.clear_cancel"),
			type: "warning",
		});

		if (!yes) return;

		emit(LISTEN_KEY.CLEAR_HISTORY);
	};

	return (
		<ProList
			header={t("preference.history.history.title")}
			footer={
				<Button block danger onClick={handleClear}>
					{t("preference.history.history.button.clear")}
				</Button>
			}
		>
			<ProListItem
				title={t("preference.history.history.label.duration")}
				description={t("preference.history.history.hints.duration")}
			>
				<Space.Compact>
					<InputNumber
						min={0}
						rootClassName="w-70"
						value={history.duration}
						onChange={(value) => {
							clipboardStore.history.duration = value ?? 0;
						}}
					/>

					<EcoSelect
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
