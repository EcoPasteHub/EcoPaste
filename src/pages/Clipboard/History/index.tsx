import { appWindow } from "@tauri-apps/api/window";

const ClipboardWindow = () => {
	useMount(() => {
		frostedWindow();
	});

	return (
		<div className="h-screen" onMouseDown={() => appWindow.startDragging()}>
			ClipboardWindow
		</div>
	);
};

export default ClipboardWindow;
