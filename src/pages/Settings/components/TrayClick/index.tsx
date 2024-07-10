import type { GlobalStore } from "@/types/store";
import { Flex, Segmented } from "antd";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: GlobalStore["trayClick"];
}

const TrayClick = () => {
	const { trayClick } = useSnapshot(globalStore);

	const options: Option[] = [
		{
			label: "无操作",
			value: "none",
		},
		{
			label: "显示应用",
			value: "show",
		},
	];

	return (
		isWin() && (
			<Flex align="center">
				<span>单击托盘：</span>
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
