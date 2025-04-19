import ProListItem from "@/components/ProListItem";
import { modifierKeys } from "@/components/ProShortcut/keyboard";
import { Select, Space, Switch } from "antd";
import type { DefaultOptionType } from "antd/es/select";
import { useSnapshot } from "valtio";

const QuickPaste = () => {
	const { shortcut } = useSnapshot(globalStore);
	const { t } = useTranslation();

	const options: DefaultOptionType[] = modifierKeys.map((item) => {
		const { tauriKey, symbol } = item;

		return {
			label: symbol,
			value: tauriKey,
			disabled: globalStore.shortcut.quickPaste.value === tauriKey,
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
