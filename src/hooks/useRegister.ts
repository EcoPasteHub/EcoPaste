import {
	type ShortcutHandler,
	isRegistered,
	register,
	unregister,
} from "@tauri-apps/plugin-global-shortcut";

export const useRegister = (
	handler: ShortcutHandler,
	deps: Array<string | undefined>,
) => {
	const [oldKey, setOldKey] = useState<string>();

	useAsyncEffect(async () => {
		const [key] = deps;

		if (oldKey) {
			const registered = await isRegistered(oldKey);

			if (registered) {
				await unregister(oldKey);
			}
		}

		if (key) {
			await register(key, (event) => {
				if (event.state === "Released") return;

				handler(event);
			});
		}

		setOldKey(key);
	}, deps);

	useUnmount(() => {
		const [key] = deps;

		if (!key) return;

		unregister(key);
	});
};
