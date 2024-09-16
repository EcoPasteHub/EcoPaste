import EcoSelect from "@/components/EcoSelect";
import ProListItem from "@/components/ProListItem";
import { modifierKeys } from "@/components/ProShortcut/keys";
import { Space, Switch } from "antd";
import type { DefaultOptionType } from "antd/es/select";
import { useSnapshot } from "valtio";

const QuickPaste = () => {
	const { shortcut } = useSnapshot(globalStore);
	const { t } = useTranslation();

	const options: DefaultOptionType[] = modifierKeys.map((item) => {
		const { key, symbol, macosSymbol } = item;

		return {
			label: isMac() ? macosSymbol : symbol,
			value: key,
		};
	});

	return (
		<ProListItem
			title={t("preference.shortcut.shortcut.label.quick_paste")}
			description={t("preference.shortcut.shortcut.hints.quick_paste")}
		>
			<Switch
				value={shortcut.quickPaste.enable}
				onChange={(value) => {
					globalStore.shortcut.quickPaste.enable = value;
				}}
			/>

			<Space>
				<EcoSelect
					options={options}
					value={shortcut.quickPaste.value}
					disabled={!shortcut.quickPaste.enable}
					onChange={(value) => {
						globalStore.shortcut.quickPaste.value = value;
					}}
				/>

				<span>1~9</span>
			</Space>
		</ProListItem>
	);
};

export default QuickPaste;
