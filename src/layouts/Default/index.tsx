import { Flex } from "antd";

const DefaultLayout = () => {
	return (
		<Flex className="h-screen">
			<Flex
				vertical
				align="center"
				justify="space-between"
				className="h-full pt-48 pb-32 transition w-68"
			>
				<Flex vertical gap="large">
					{routes[0].children?.map((item) => {
						const { path, meta } = item;

						const { title } = meta!;

						return (
							<Link key={title} to={path}>
								<Flex vertical align="center" gap={4}>
									{/* <Icon name={icon} size={path === "/keyboard" ? 24 : 26} /> */}
									<span>{title}</span>
								</Flex>
							</Link>
						);
					})}
				</Flex>
				<div>主题</div>
			</Flex>

			<div className="h-full">
				<Outlet />
			</div>
		</Flex>
	);
};

export default DefaultLayout;
