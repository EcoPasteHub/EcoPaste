import ProListItem from "@/components/ProListItem";
import { InputNumber } from "antd";
import { useSnapshot } from "valtio";

const MaxCount = () => {
	const { history } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	return (
		<ProListItem
			title={t("preference.history.history.label.max_count")}
			description={t("preference.history.history.hints.max_count")}
		>
			<InputNumber
				value={history.maxCount}
				min={0}
				addonAfter={t("preference.history.history.label.max_count_unit")}
				className="w-120"
				onChange={(value) => {
					clipboardStore.history.maxCount = value ?? 0;
				}}
			/>
		</ProListItem>
	);
};

export default MaxCount;
