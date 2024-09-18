import ProListItem from "@/components/ProListItem";
import { modifierKeys } from "@/components/ProShortcut/keys";
import { Select, Space, Switch } from "antd";
import type { DefaultOptionType } from "antd/es/select";
import {} from "lodash-es";
import { useSnapshot } from "valtio";

const QuickPaste = () => {
	const { shortcut } = useSnapshot(globalStore);
	const { t } = useTranslation();

	const options: DefaultOptionType[] = modifierKeys.map((item) => {
		const { key, symbol, macosSymbol } = item;

		return {
			label: isMac() ? macosSymbol : symbol,
			value: key,
			disabled: globalStore.shortcut.quickPaste.value === key,
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
				<Select
					mode="multiple"
					maxCount={2}
					showSearch={false}
					options={options}
					value={shortcut.quickPaste.value?.split("+")}
					disabled={!shortcut.quickPaste.enable}
					onChange={(value) => {
						globalStore.shortcut.quickPaste.value = value.join("+");
					}}
				/>

				<span>1~9</span>
			</Space>
		</ProListItem>
	);
};

export default QuickPaste;
