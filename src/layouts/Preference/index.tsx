import Icon from "@/components/Icon";
import UpdateApp from "@/components/UpdateApp";
import About from "@/pages/About";
import Backup from "@/pages/Backup";
import ClipboardSettings from "@/pages/Clipboard/Settings";
import General from "@/pages/General";
import History from "@/pages/History";
import Shortcut from "@/pages/Shortcut";
import { emit } from "@tauri-apps/api/event";
import { Flex } from "antd";
import clsx from "clsx";
import { MacScrollbar } from "mac-scrollbar";
import { useSnapshot } from "valtio";

const PreferenceLayout = () => {
	const { t } = useTranslation();
	const { app, shortcut, appearance } = useSnapshot(globalStore);
	const [activeKey, setActiveKey] = useState("clipboard");
	const contentRef = useRef<HTMLElement>(null);

	useTray();

	useMount(async () => {
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
				content: <ClipboardSettings />,
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
		<Flex className="h-screen">
			<Flex
				data-tauri-drag-region
				vertical
				gap="small"
				className={clsx("h-full w-200 p-12", [
					isMac() ? "pt-32" : "bg-color-1",
				])}
			>
				{menuItems.map((item) => {
					const { key, label, icon } = item;

					return (
						<Flex
							key={key}
							align="center"
							gap="small"
							className={clsx(
								"cursor-pointer rounded-8 p-12 p-r-0 text-color-2 transition hover:bg-color-4",
								{
									"bg-primary! text-white!": activeKey === key,
								},
							)}
							onClick={() => handleMenuClick(key)}
						>
							<Icon name={icon} size={20} />

							<span className="font-bold">{label}</span>
						</Flex>
					);
				})}
			</Flex>

			<MacScrollbar
				data-tauri-drag-region
				ref={contentRef}
				skin={appearance.isDark ? "dark" : "light"}
				className="h-full flex-1 bg-color-2 p-16"
			>
				{menuItems.map((item) => {
					const { key, content } = item;

					return (
						<div key={key} hidden={key !== activeKey}>
							{content}
						</div>
					);
				})}
			</MacScrollbar>

			<UpdateApp />
		</Flex>
	);
};

export default PreferenceLayout;
