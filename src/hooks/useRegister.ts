import {
	type ShortcutHandler,
	register,
	unregister,
} from "@tauri-apps/api/globalShortcut";

export const useRegister = (
	handler: ShortcutHandler,
	deps: Array<string | undefined>,
) => {
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

	useUnmount(() => {
		const [key] = deps;

		if (!key) return;

		unregister(key);
	});
};
