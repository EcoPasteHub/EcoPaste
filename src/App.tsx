import { listen } from "@tauri-apps/api/event";
import { ConfigProvider, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import { isEqual } from "arcdash";
import { RouterProvider } from "react-router-dom";
import { useSnapshot } from "valtio";

const { defaultAlgorithm, darkAlgorithm } = theme;

const App = () => {
	const { isDark } = useTheme();
	const { platform } = useSnapshot(globalStore);

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
	});

	useEventListener("contextmenu", (event) => {
		if (isDev()) return;

		event.preventDefault();
	});

	return (
		platform && (
			<ConfigProvider
				locale={zhCN}
				theme={{
					algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
				}}
			>
				<RouterProvider router={router} />
			</ConfigProvider>
		)
	);
};

export default App;
