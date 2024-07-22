import { HappyProvider } from "@ant-design/happy-work-theme";
import { listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/api/shell";
import { ConfigProvider, theme } from "antd";
import { isEqual } from "arcdash";
import { RouterProvider } from "react-router-dom";
import { useSnapshot } from "valtio";

const { defaultAlgorithm, darkAlgorithm } = theme;

const App = () => {
	const { isDark } = useTheme();
	const { language } = useSnapshot(globalStore);

	useMount(() => {
		generateColorVars();

		listen(LISTEN_KEY.GLOBAL_STORE_CHANGED, ({ payload }) => {
			if (isEqual(globalStore, payload)) return;

			Object.assign(globalStore, payload);
		});

		listen(LISTEN_KEY.CLIPBOARD_STORE_CHANGED, ({ payload }) => {
			if (isEqual(clipboardStore, payload)) return;

			Object.assign(clipboardStore, payload);
		});

		initDatabase();
	});

	useEffect(() => {
		i18n.changeLanguage(language);

		setLocale(language);
	}, [language]);

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

	useEventListener("keydown", (event) => {
		if (event.code === "Escape") {
			event.preventDefault();

			hideWindow();
		}
	});

	return (
		<ConfigProvider
			locale={getAntdLocale(language)}
			theme={{
				algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
			}}
		>
			<HappyProvider>
				<RouterProvider router={router} />
			</HappyProvider>
		</ConfigProvider>
	);
};

export default App;
