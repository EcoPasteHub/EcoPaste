import Icon from "@/components/Icon";
import Tray from "@/components/Tray";
import UpdateApp from "@/components/UpdateApp";
import About from "@/pages/About";
import Backup from "@/pages/Backup";
import ClipboardSettings from "@/pages/Clipboard/Settings";
import General from "@/pages/General";
import History from "@/pages/History";
import Shortcut from "@/pages/Shortcut";
import { emit } from "@tauri-apps/api/event";
import { Flex, Tabs, type TabsProps } from "antd";
import clsx from "clsx";
import { subscribe, useSnapshot } from "valtio";
import styles from "./index.module.scss";

const PreferenceLayout = () => {
	const { t } = useTranslation();
	const { app, shortcut } = useSnapshot(globalStore);
	const [activeKey, setActiveKey] = useState("clipboard");

	useMount(async () => {
		const autostart = await isAutostart();

		if (!autostart && !app.silentStart) {
			showWindow();
		}

		// 监听全局配置项变化
		subscribe(globalStore, handleStoreChanged);

		// 监听剪贴板配置项变化
		subscribe(clipboardStore, handleStoreChanged);
	});

	// 监听快捷键切换窗口显隐
	useRegister(toggleWindowVisible, [shortcut.preference]);

	const tabItems: TabsProps["items"] = [
		{
			key: "clipboard",
			label: t("preference.menu.title.clipboard"),
			icon: "i-lucide:clipboard-list",
			children: <ClipboardSettings />,
		},
		{
			key: "history",
			label: t("preference.menu.title.history"),
			icon: "i-lucide:history",
			forceRender: true,
			children: <History />,
		},
		{
			key: "general",
			label: t("preference.menu.title.general"),
			icon: "i-lucide:bolt",
			forceRender: true,
			children: <General />,
		},
		{
			key: "shortcut",
			label: t("preference.menu.title.shortcut"),
			icon: "i-lucide:keyboard",
			children: <Shortcut />,
		},
		{
			key: "backup",
			label: t("preference.menu.title.backup"),
			icon: "i-lucide:database-backup",
			children: <Backup />,
		},
		{
			key: "about",
			label: t("preference.menu.title.about"),
			icon: "i-lucide:info",
			children: <About />,
		},
	];

	const renderTabBar: TabsProps["renderTabBar"] = () => {
		return (
			<Flex
				data-tauri-drag-region
				vertical
				gap="small"
				className={clsx("h-full w-200 p-12", [isMac() ? "pt-32" : "bg-1"])}
				onClick={(event) => event.stopPropagation()}
			>
				{tabItems.map((item) => {
					const { key, label, icon } = item;

					return (
						<Flex
							key={key}
							align="center"
							gap="small"
							className={clsx(
								"color-2 cursor-pointer rounded-8 p-12 p-r-0 transition hover:bg-4",
								{
									"bg-primary! text-white!": activeKey === key,
								},
							)}
							onClick={() => handleTabItemClick(key)}
						>
							<Icon name={icon as string} size={20} />

							<span className="font-bold">{label}</span>
						</Flex>
					);
				})}
			</Flex>
		);
	};

	// 配置项变化通知其它窗口和本地存储
	const handleStoreChanged = async () => {
		emit(LISTEN_KEY.STORE_CHANGED, { globalStore, clipboardStore });

		saveStore();
	};

	// 切换选项卡
	const handleTabItemClick = (key: string) => {
		setActiveKey(key);

		raf(() => {
			const element = document.querySelector(".ant-tabs-content-holder");

			element?.scrollTo({ top: 0, behavior: "smooth" });
		});
	};

	return (
		<>
			<Tabs
				animated
				activeKey={activeKey}
				tabPosition="left"
				items={tabItems}
				className={styles.root}
				renderTabBar={renderTabBar}
			/>

			<Tray />

			<UpdateApp />
		</>
	);
};

export default PreferenceLayout;
