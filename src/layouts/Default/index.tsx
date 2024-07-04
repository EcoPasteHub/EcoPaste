import Icon from "@/components/Icon";
import Update from "@/components/Update";
import { emit, listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/api/shell";
import { Flex } from "antd";
import clsx from "clsx";
import { subscribe, useSnapshot } from "valtio";

const DefaultLayout = () => {
	const { pathname } = useLocation();
	const navigate = useNavigate();
	const { isDark, wakeUpKey } = useSnapshot(globalStore);

	const [sidebarClassName, setSidebarClassName] = useState("pt-48");

	useMount(async () => {
		if (await isWin()) {
			setSidebarClassName("pt-32");
		}

		createWindow("/clipboard-history");

		listen(LISTEN_KEY.GITHUB, () => {
			open(GITHUB_LINK);
		});

		listen(LISTEN_KEY.ABOUT, () => {
			showWindow();

			navigate("/about");
		});

		subscribe(globalStore, () => {
			emit(LISTEN_KEY.GLOBAL_STORE_CHANGED, globalStore);
		});

		subscribe(clipboardStore, () => {
			emit(LISTEN_KEY.CLIPBOARD_STORE_CHANGED, clipboardStore);
		});
	});

	useRegister(toggleWindowVisible, [wakeUpKey]);

	return (
		<Flex className="h-screen">
			<Flex
				data-tauri-drag-region
				vertical
				align="center"
				justify="space-between"
				className={clsx(
					"color-2 h-full w-90 bg-2 pb-32 transition",
					sidebarClassName,
				)}
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
					name={isDark ? "i-iconamoon:mode-light" : "i-iconamoon:mode-dark"}
					onMouseDown={() => toggleTheme(isDark ? "light" : "dark")}
				/>
			</Flex>

			<div
				data-tauri-drag-region
				className="h-full flex-1 overflow-auto bg-1 p-16 transition"
			>
				<Outlet />
			</div>

			<Update />
		</Flex>
	);
};

export default DefaultLayout;
