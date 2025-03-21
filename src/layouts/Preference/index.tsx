import Icon from "@/components/Icon";
import Scrollbar from "@/components/Scrollbar";
import Tray from "@/components/Tray";
import UpdateApp from "@/components/UpdateApp";
import About from "@/pages/About";
import Backup from "@/pages/Backup";
import ClipboardSettings from "@/pages/Clipboard/Settings";
import General from "@/pages/General";
import History from "@/pages/History";
import Shortcut from "@/pages/Shortcut";
import { emit } from "@tauri-apps/api/event";
import { Layout, Menu } from "antd";
import Sider from "antd/es/layout/Sider";
import { Content } from "antd/lib/layout/layout";
import { useSnapshot } from "valtio";
import styles from "./index.module.scss";

interface TabItem {
	key: string;
	label: string;
	icon: string;
	children: React.ReactNode;
}

const PreferenceLayout = () => {
	const { t } = useTranslation();
	const { app, shortcut } = useSnapshot(globalStore);
	const [activeKey, setActiveKey] = useState("clipboard");

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

	const tabItems: TabItem[] = [
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
			children: <History />,
		},
		{
			key: "general",
			label: t("preference.menu.title.general"),
			icon: "i-lucide:bolt",
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

	// 配置项变化通知其它窗口和本地存储
	const handleStoreChanged = async () => {
		emit(LISTEN_KEY.STORE_CHANGED, { globalStore, clipboardStore });

		saveStore();
	};

	const menuItems = tabItems.map((item) => ({
		key: item.key,
		label: <span className="font-bold">{item.label}</span>,
		icon: <Icon name={item.icon as string} size={20} />,
	}));

	// 根据 activeKey 获取当前要显示的内容
	const getCurrentContent = () => {
		const currentTab = tabItems.find((item) => item.key === activeKey);
		return currentTab?.children;
	};

	// 处理菜单点击事件
	const handleMenuClick = ({ key }: { key: string }) => {
		setActiveKey(key);
	};

	return (
		<>
			<Layout className={styles.root}>
				<Sider>
					<Menu
						className="h-full p-t-24"
						mode="inline"
						items={menuItems}
						selectedKeys={[activeKey]}
						onClick={handleMenuClick}
					/>
				</Sider>
				<Content className="p-r-2 py-4">
					<Scrollbar className="h-full p-10">{getCurrentContent()}</Scrollbar>
				</Content>
			</Layout>

			<Tray />

			<UpdateApp />
		</>
	);
};

export default PreferenceLayout;
