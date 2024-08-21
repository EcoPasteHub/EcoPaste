import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["window"]["position"];
}

const WindowPosition = () => {
	const { window } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t(
				"preference.clipboard.window_settings.label.window_position_remember",
			),
			value: "remember",
		},
		{
			label: t(
				"preference.clipboard.window_settings.label.window_position_follow",
			),
			value: "follow",
		},
		{
			label: t(
				"preference.clipboard.window_settings.label.window_position_center",
			),
			value: "center",
		},
	];

	return (
		<ProSelect
			title={t("preference.clipboard.window_settings.label.window_position")}
			value={window.position}
			options={options}
			onChange={(value) => {
				clipboardStore.window.position = value;
			}}
		/>
	);
};

export default WindowPosition;
