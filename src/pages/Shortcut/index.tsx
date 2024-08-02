import ProList from "@/components/ProList";
import ProShortcut from "@/components/ProShortcut";

const Shortcut = () => {
	return (
		<ProList header="快捷键">
			<ProShortcut
				title="唤醒剪贴板"
				defaultValue={clipboardStore.wakeUpKey}
				onChange={(value) => {
					clipboardStore.wakeUpKey = value;
				}}
			/>

			<ProShortcut
				title="唤醒偏好设置"
				defaultValue={globalStore.wakeUpKey}
				onChange={(value) => {
					globalStore.wakeUpKey = value;
				}}
			/>
		</ProList>
	);
};

export default Shortcut;
