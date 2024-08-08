import ProList from "@/components/ProList";
import ProShortcut from "@/components/ProShortcut";
import { useSnapshot } from "valtio";

const Shortcut = () => {
	const { shortcut } = useSnapshot(globalStore);

	return (
		<ProList header="快捷键">
			<ProShortcut
				title="打开剪贴板窗口"
				defaultValue={shortcut.clipboard}
				onChange={(value) => {
					globalStore.shortcut.clipboard = value;
				}}
			/>

			<ProShortcut
				title="打开偏好设置窗口"
				defaultValue={shortcut.preference}
				onChange={(value) => {
					globalStore.shortcut.preference = value;
				}}
			/>

			<ProShortcut
				title="自动粘贴纯文本"
				description="自动粘贴纯文本或者OCR文本"
				defaultValue={shortcut.paste}
				onChange={(value) => {
					globalStore.shortcut.paste = value;
				}}
			/>
		</ProList>
	);
};

export default Shortcut;
