import {
	type ShortcutHandler,
	register,
	unregister,
} from "@tauri-apps/api/globalShortcut";

export const useRegister = (handler: ShortcutHandler, deps: string[]) => {
	const [oldKey, setOldKey] = useState(deps[0]);

	useAsyncEffect(async () => {
		const [key] = deps;

		if (oldKey) {
			await unregister(oldKey);
		}

		if (key) {
			await register(key, handler);
		}

		setOldKey(key);
	}, deps);
};
