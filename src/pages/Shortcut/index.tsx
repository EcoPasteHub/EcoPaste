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
		</ProList>
	);
};

export default Shortcut;
