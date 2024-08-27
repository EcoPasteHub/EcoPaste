import { HappyProvider } from "@ant-design/happy-work-theme";
import { open } from "@tauri-apps/api/shell";
import { appWindow } from "@tauri-apps/api/window";
import { ConfigProvider, theme } from "antd";
import { RouterProvider } from "react-router-dom";
import { useSnapshot } from "valtio";
import { subscribeKey } from "valtio/utils";
const { defaultAlgorithm, darkAlgorithm } = theme;
import { isString } from "arcdash";
import { error } from "tauri-plugin-log-api";

const App = () => {
	const { appearance } = useSnapshot(globalStore);

	useMount(() => {
		handleSystemThemeChanged();

		appWindow.onThemeChanged(handleSystemThemeChanged);

		initDatabase();

		generateColorVars();

		watchKey(globalStore.appearance, "language", (value = "zh-CN") => {
			i18n.changeLanguage(value);

			setLocale(value);
		});

		subscribeKey(globalStore.appearance, "theme", async (value) => {
			let nextTheme = value;

			if (isWin()) {
				const yes = await ask("切换主题需要重启 app 才能生效！", {
					okLabel: "重启",
					cancelLabel: "取消",
					type: "warning",
				});

				if (!yes) return;
			}

			if (nextTheme === "auto") {
				nextTheme = (await appWindow.theme()) ?? "light";
			}

			globalStore.appearance.isDark = nextTheme === "dark";

			setTheme(value);
		});

		watchKey(globalStore.appearance, "isDark", (value) => {
			if (value) {
				document.documentElement.classList.add("dark");
			} else {
				document.documentElement.classList.remove("dark");
			}
		});
	});

	const handleSystemThemeChanged = async () => {
		if (globalStore.appearance.theme !== "auto") return;

		const systemTheme = await appWindow.theme();

		globalStore.appearance.isDark = systemTheme === "dark";
	};

	useEventListener("contextmenu", (event) => {
		if (isDev()) return;

		event.preventDefault();
	});

	useEventListener("click", (event) => {
		const link = (event.target as HTMLElement).closest("a");

		if (!link) return;

		const { href, target } = link;

		if (target === "_blank") return;

		event.preventDefault();

		if (!isURL(href)) return;

		open(href);
	});

	useOSKeyPress(["esc", "meta.w"], hideWindow);

	useEventListener("unhandledrejection", ({ reason }) => {
		const message = isString(reason) ? reason : JSON.stringify(reason);

		error(message);
	});

	return (
		<ConfigProvider
			locale={getAntdLocale(appearance.language)}
			theme={{
				algorithm: appearance.isDark ? darkAlgorithm : defaultAlgorithm,
			}}
		>
			<HappyProvider>
				<RouterProvider router={router} />
			</HappyProvider>
		</ConfigProvider>
	);
};

export default App;
