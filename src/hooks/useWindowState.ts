import { PhysicalPosition, PhysicalSize } from "@tauri-apps/api/dpi";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { exists, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";

const appWindow = getCurrentWebviewWindow();
const { label } = appWindow;

export const useWindowState = () => {
	const state = useReactive<Partial<PhysicalPosition & PhysicalSize>>({});

	useMount(() => {
		appWindow.onMoved(({ payload }) => {
			Object.assign(state, payload);
		});

		appWindow.onResized(({ payload }) => {
			Object.assign(state, payload);
		});
	});

	useTauriFocus({
		onBlur() {
			saveState();
		},
	});

	const getSavedStates = async () => {
		const path = await saveWindowStatePath();

		const existed = await exists(path);

		if (!existed) return {};

		const states = await readTextFile(path);

		return JSON.parse(states);
	};

	const saveState = async () => {
		const path = await saveWindowStatePath();

		const states = await getSavedStates();

		states[label] = state;

		return writeTextFile(path, JSON.stringify(states, null, 2));
	};

	const restoreState = async () => {
		const states = await getSavedStates();

		Object.assign(state, states[label]);

		const { x, y, width, height } = state;

		if (x && y) {
			appWindow.setPosition(new PhysicalPosition(x, y));
		}

		if (width && height) {
			appWindow.setSize(new PhysicalSize(width, height));
		}
	};

	return {
		saveState,
		restoreState,
	};
};
