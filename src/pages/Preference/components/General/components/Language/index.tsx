import ProSelect from "@/components/ProSelect";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useSnapshot } from "valtio";

interface Option {
	label: string;
	value: string;
}

const Language = () => {
	const { appearance } = useSnapshot(globalStore);
	const { t } = useTranslation();

	useImmediateKey(globalStore.appearance, "language", () => {
		const appWindow = getCurrentWebviewWindow();

		raf(() => {
			appWindow.setTitle(t("preference.title"));
		});
	});

	const options: Option[] = [
		{
			label: "简体中文",
			value: LANGUAGE.ZH_CN,
		},
		{
			label: "繁體中文",
			value: LANGUAGE.ZH_TW,
		},
		{
			label: "English",
			value: LANGUAGE.EN_US,
		},
		{
			label: "日本語",
			value: LANGUAGE.JA_JP,
		},
		{
			label: "Tiếng Việt",
			value: LANGUAGE.VI_VN,
		},
	];

	return (
		<ProSelect
			title={t("preference.settings.appearance_settings.label.language")}
			value={appearance.language}
			options={options}
			onChange={(value) => {
				globalStore.appearance.language = value;
			}}
		/>
	);
};

export default Language;
