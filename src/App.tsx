import { HappyProvider } from "@ant-design/happy-work-theme";
import { listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/api/shell";
import { ConfigProvider, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import { isEqual } from "arcdash";
import { RouterProvider } from "react-router-dom";

const { defaultAlgorithm, darkAlgorithm } = theme;

const App = () => {
	const { isDark } = useTheme();

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

	useEventListener("contextmenu", (event) => {
		if (isDev()) return;

		event.preventDefault();
	});

	useEventListener("click", (event) => {
		const target = event.target as HTMLElement;

		const link = target.closest("a");

		// 如果是 a 元素，并且 href 是链接，且 target 不是 "_blank"，那么就调用 open 方法在默认浏览器中打开
		if (!link || !isURL(link.href) || link.target === "_blank") return;

		event.preventDefault();

		open(link.href);
	});

	return (
		<ConfigProvider
			locale={zhCN}
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
