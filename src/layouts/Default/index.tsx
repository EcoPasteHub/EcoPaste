import { ConfigProvider, Flex } from "antd";
import zhCN from "antd/locale/zh_CN";
import clsx from "clsx";

const DefaultLayout = () => {
	const { pathname } = useLocation();

	return (
		<ConfigProvider locale={zhCN}>
			<Flex className="h-screen">
				<Flex
					data-tauri-drag-region
					vertical
					align="center"
					justify="space-between"
					className="color-2 h-full w-100 bg-2 pt-48 pb-32 transition"
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
										<i className={clsx(icon, "text-22")} />
										<span>{title}</span>
									</Flex>
								</Link>
							);
						})}
					</Flex>
					<div>主题</div>
				</Flex>

				<div
					data-tauri-drag-region
					className="h-full flex-1 overflow-auto p-16"
				>
					<Outlet />
				</div>
			</Flex>
		</ConfigProvider>
	);
};

export default DefaultLayout;
