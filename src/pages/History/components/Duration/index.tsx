import ProListItem from "@/components/ProListItem";
import { InputNumber } from "antd";
import { useSnapshot } from "valtio";

const Duration = () => {
	const { history } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	return (
		<ProListItem
			title={t("preference.history.history.label.duration")}
			description={t("preference.history.history.hints.duration")}
		>
			<InputNumber
				value={history.duration}
				min={0}
				addonAfter={t("preference.history.history.label.duration_unit")}
				className="w-120"
				onChange={(value) => {
					clipboardStore.history.duration = value ?? 0;
				}}
			/>
		</ProListItem>
	);
};

export default Duration;
