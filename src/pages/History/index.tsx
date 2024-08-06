import ProList from "@/components/ProList";
import ProSelect from "@/components/ProSelect";
import { emit } from "@tauri-apps/api/event";
import { Button } from "antd";
import { useSnapshot } from "valtio";

const History = () => {
	const { historyDuration } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options = [
		{
			label: t("preference.history.label.unlimited"),
			value: 0,
		},
		{
			label: t("preference.history.label.day"),
			value: 1,
		},
		{
			label: t("preference.history.label.week"),
			value: 7,
		},
		{
			label: t("preference.history.label.month"),
			value: 30,
		},
		{
			label: t("preference.history.label.half_year"),
			value: 180,
		},
		{
			label: t("preference.history.label.year"),
			value: 365,
		},
	];

	const handleClear = async () => {
		const yes = await ask(t("preference.history.clear.hints.clear_confirm"), {
			title: t("preference.history.clear.title"),
			okLabel: t("preference.history.clear.button.confirm"),
			cancelLabel: t("preference.history.clear.button.cancel"),
			type: "warning",
		});

		if (!yes) return;

		emit(LISTEN_KEY.CLEAR_HISTORY);
	};

	return (
		<ProList
			header={t("preference.history.title")}
			footer={
				<Button block danger onClick={handleClear}>
					{t("preference.history.button")}
				</Button>
			}
		>
			<ProSelect
				title={t("preference.history.duration")}
				value={historyDuration}
				options={options}
				onChange={(value) => {
					clipboardStore.historyDuration = value;
				}}
			/>
		</ProList>
	);
};

export default History;
