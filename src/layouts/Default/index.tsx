import Icon from "@/components/Icon";
import { listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/api/shell";
import { ConfigProvider, Flex, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import clsx from "clsx";
import { useSnapshot } from "valtio";

const { defaultAlgorithm, darkAlgorithm } = theme;

const DefaultLayout = () => {
	const { pathname } = useLocation();
	const { isDark } = useSnapshot(store);

	useMount(() => {
		listen("github", () => {
			open("https://github.com/ayangweb/EcoCopy");
		});
	});

	return (
		<ConfigProvider
			locale={zhCN}
			theme={{
				algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
			}}
		>
			<Flex className="h-screen">
				<Flex
					data-tauri-drag-region
					vertical
					align="center"
					justify="space-between"
					className="color-2 h-full w-90 bg-2 pt-48 pb-32 transition"
				>
					<Flex vertical gap="large">
						{routes[0].children?.map((item) => {
							const { path, meta = {} } = item;

							const { title, icon } = meta;

							return (
								<Link
									key={title}
									to={path}
									className={clsx("hover:text-primary", {
										"text-primary": pathname === path,
									})}
								>
									<Flex vertical align="center" gap={4}>
										<Icon name={icon} size={22} />
										<span>{title}</span>
									</Flex>
								</Link>
							);
						})}
					</Flex>
					<Icon
						hoverable
						size={24}
						name={isDark ? "i-iconamoon-mode-light" : "i-iconamoon-mode-dark"}
						onMouseDown={() => toggleTheme(isDark ? "light" : "dark")}
					/>
				</Flex>

				<div
					data-tauri-drag-region
					className="h-full flex-1 overflow-auto bg-1 p-16 transition"
				>
					<Outlet />
				</div>
			</Flex>
		</ConfigProvider>
	);
};

export default DefaultLayout;
