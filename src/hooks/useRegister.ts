import {
	type ShortcutHandler,
	isRegistered,
	register,
	unregister,
} from "@tauri-apps/plugin-global-shortcut";
import { castArray } from "lodash-es";

export const useRegister = (
	handler: ShortcutHandler,
	deps: Array<string | string[] | undefined>,
) => {
	const [oldShortcuts, setOldShortcuts] = useState(deps[0]);

	useAsyncEffect(async () => {
		const [shortcuts] = deps;

		if (!shortcuts) return;

		for await (const shortcut of castArray(oldShortcuts)) {
			const registered = await isRegistered(shortcut);

			if (registered) {
				await unregister(shortcut);
			}
		}

		await register(shortcuts, (event) => {
			if (event.state === "Released") return;

			handler(event);
		});

		setOldShortcuts(shortcuts);
	}, deps);

	useUnmount(() => {
		const [shortcuts] = deps;

		if (!shortcuts) return;

		unregister(shortcuts);
	});
};
