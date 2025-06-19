import Scrollbar from "@/components/Scrollbar";
import UnoIcon from "@/components/UnoIcon";
import UpdateApp from "@/components/UpdateApp";
import { emit } from "@tauri-apps/api/event";
import { Flex } from "antd";
import clsx from "clsx";
import { useSnapshot } from "valtio";
import About from "./components/About";
import Backup from "./components/Backup";
import Clipboard from "./components/Clipboard";
import General from "./components/General";
import History from "./components/History";
import Shortcut from "./components/Shortcut";

const Preference = () => {
	const { t } = useTranslation();
	const { app, shortcut, appearance } = useSnapshot(globalStore);
	const [activeKey, setActiveKey] = useState("clipboard");
	const contentRef = useRef<HTMLElement>(null);

	const { createTray } = useTray();

	useMount(async () => {
		createTray();

		const autostart = await isAutostart();

		if (!autostart && !app.silentStart) {
			showWindow();
		}
	});

	// 监听全局配置项变化
	useSubscribe(globalStore, () => handleStoreChanged());

	// 监听剪贴板配置项变化
	useSubscribe(clipboardStore, () => handleStoreChanged());

	// 监听快捷键切换窗口显隐
	useRegister(toggleWindowVisible, [shortcut.preference]);

	// 配置项变化通知其它窗口和本地存储
	const handleStoreChanged = () => {
		emit(LISTEN_KEY.STORE_CHANGED, { globalStore, clipboardStore });

		saveStore();
	};

	const menuItems = useCreation(() => {
		return [
			{
				key: "clipboard",
				label: t("preference.menu.title.clipboard"),
				icon: "i-lucide:clipboard-list",
				content: <Clipboard />,
			},
			{
				key: "history",
				label: t("preference.menu.title.history"),
				icon: "i-lucide:history",
				content: <History />,
			},
			{
				key: "general",
				label: t("preference.menu.title.general"),
				icon: "i-lucide:bolt",
				content: <General />,
			},
			{
				key: "shortcut",
				label: t("preference.menu.title.shortcut"),
				icon: "i-lucide:keyboard",
				content: <Shortcut />,
			},
			{
				key: "backup",
				label: t("preference.menu.title.backup"),
				icon: "i-lucide:database-backup",
				content: <Backup />,
			},
			{
				key: "about",
				label: t("preference.menu.title.about"),
				icon: "i-lucide:info",
				content: <About />,
			},
		];
	}, [appearance.language]);

	const handleMenuClick = (key: string) => {
		setActiveKey(key);

		raf(() => {
			contentRef.current?.scrollTo({ top: 0, behavior: "smooth" });
		});
	};

	return (
		<div
			className={clsx("h-screen", {
				"rounded-2.5": !isWin,
				"b b-color-2": appearance.isDark,
			})}
		>
			<Flex className="h-full">
				<Flex
					data-tauri-drag-region
					vertical
					gap="small"
					className={clsx("h-full w-50 p-3", [isMac ? "pt-8" : "bg-color-1"])}
				>
					{menuItems.map((item) => {
						const { key, label, icon } = item;

						return (
							<Flex
								key={key}
								align="center"
								gap="small"
								className={clsx(
									"cursor-pointer rounded-lg p-3 p-r-0 text-color-2 transition hover:bg-color-4",
									{
										"bg-primary! text-white!": activeKey === key,
									},
								)}
								onClick={() => handleMenuClick(key)}
							>
								<UnoIcon name={icon} size={20} />

								<span className="font-bold">{label}</span>
							</Flex>
						);
					})}
				</Flex>

				<div
					className={clsx("flex-1 bg-color-2 py-3", {
						"rounded-r-2.5": !isWin,
					})}
				>
					<Scrollbar
						data-tauri-drag-region
						ref={contentRef}
						offset={3}
						className="h-full px-4"
					>
						{menuItems.map((item) => {
							const { key, content } = item;

							return (
								<div key={key} hidden={key !== activeKey}>
									{content}
								</div>
							);
						})}
					</Scrollbar>
				</div>

				<UpdateApp />
			</Flex>
		</div>
	);
};

export default Preference;
