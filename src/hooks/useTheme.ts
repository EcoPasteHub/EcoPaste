import type { Theme } from "@/types/store";
import { appWindow } from "@tauri-apps/api/window";
import { useSnapshot } from "valtio";

export const useTheme = () => {
	const { theme } = useSnapshot(globalStore);
	const { t } = useTranslation();
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
			const yes = await ask(t("component.use_theme.hints.reboot_confirm"), {
				okLabel: t("component.use_theme.button.confirm_reboot"),
				cancelLabel: t("component.use_theme.button.cancel"),
				type: "warning",
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
