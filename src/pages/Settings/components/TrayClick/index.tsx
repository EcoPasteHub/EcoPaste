import type { GlobalStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: GlobalStore["trayClick"];
}

const TrayClick = () => {
	const { trayClick } = useSnapshot(globalStore);
	const { t } = useTranslation();

	const options: Option[] = [
		{
			label: t("preference.settings.basic.label.tray_click_none"),
			value: "none",
		},
		{
			label: t("preference.settings.basic.label.tray_click_show"),
			value: "show",
		},
	];

	return (
		isWin() && (
			<Flex align="center">
				<span>{t("preference.settings.basic.label.tray_click")}：</span>
				<Segmented
					value={trayClick}
					options={options}
					onChange={(value) => {
						globalStore.trayClick = value;
					}}
				/>
			</Flex>
		)
	);
};

export default TrayClick;
