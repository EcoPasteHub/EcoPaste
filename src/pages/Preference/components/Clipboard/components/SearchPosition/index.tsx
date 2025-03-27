import ProSelect from "@/components/ProSelect";
import type { ClipboardStore } from "@/types/store";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: ClipboardStore["search"]["position"];
}

const SearchPosition = () => {
	const { search } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.clipboard.search_box_settings.label.position_top"),
			value: "top",
		},
		{
			label: t(
				"preference.clipboard.search_box_settings.label.position_bottom",
			),
			value: "bottom",
		},
	];

	return (
		<ProSelect
			title={t("preference.clipboard.search_box_settings.label.position")}
			value={search.position}
			options={options}
			onChange={(value) => {
				clipboardStore.search.position = value;
			}}
		/>
	);
};

export default SearchPosition;
