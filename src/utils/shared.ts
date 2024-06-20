import type { Theme } from "@/types/store";
import { invoke } from "@tauri-apps/api";
import { ask } from "@tauri-apps/api/dialog";

/**
 * 切换主题
 */
export const toggleTheme = async (theme: Theme) => {
	if (await isWin()) {
		const yes = await ask("切换主题需要重启 app 才能生效！", {
			okLabel: "重启",
			cancelLabel: "取消",
		});

		if (!yes) return;
	}

	globalStore.theme = theme;

	invoke("plugin:theme|set_theme", { theme });
};
