import type { Theme } from "@/types/store";
import { ask } from "@tauri-apps/api/dialog";
import { appWindow } from "@tauri-apps/api/window";
import { useSnapshot } from "valtio";

export const useTheme = () => {
	const { theme } = useSnapshot(globalStore);
	const [isDark, setIsDark] = useState(false);

	useMount(() => {
		appWindow.onThemeChanged(({ payload }) => {
			if (globalStore.theme !== "auto") return;

			setIsDark(payload === "dark");
		});
	});

	useAsyncEffect(async () => {
		let value = theme;

		if (value === "auto") {
			value = (await appWindow.theme()) ?? "light";
		}

		setIsDark(value === "dark");
	}, [theme]);

	useEffect(() => {
		if (isDark) {
			document.documentElement.classList.add("dark");
		} else {
			document.documentElement.classList.remove("dark");
		}
	}, [isDark]);

	const toggleTheme = async (theme?: Theme) => {
		const nextTheme = theme ?? (isDark ? "light" : "dark");

		if (isWin()) {
			const yes = await ask("切换主题需要重启 app 才能生效！", {
				okLabel: "重启",
				cancelLabel: "取消",
			});

			if (!yes) return;
		}

		globalStore.theme = nextTheme;

		setTheme(nextTheme);
	};

	return {
		theme,
		isDark,
		toggleTheme,
	};
};
