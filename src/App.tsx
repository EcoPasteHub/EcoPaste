import { ConfigProvider, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import { RouterProvider } from "react-router-dom";
import { useSnapshot } from "valtio";

const { defaultAlgorithm, darkAlgorithm } = theme;

const App = () => {
	const { isDark } = useSnapshot(store);

	useMount(() => {
		initDatabase();

		generateColorVars();
	});

	return (
		<ConfigProvider
			locale={zhCN}
			theme={{
				algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
			}}
		>
			<RouterProvider router={router} />
		</ConfigProvider>
	);
};

export default App;
