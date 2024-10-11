import { emit } from "@tauri-apps/api/event";
import { Menu, MenuItem, PredefinedMenuItem } from "@tauri-apps/api/menu";
import { resolveResource } from "@tauri-apps/api/path";
import { TrayIcon, type TrayIconOptions } from "@tauri-apps/api/tray";
import { exit } from "@tauri-apps/plugin-process";
import { open } from "@tauri-apps/plugin-shell";

const Tray = () => {
	const navigate = useNavigate();
	const [startListen, { toggle }] = useBoolean(true);

	useMount(async () => {
		await createTrayIcon();

		// 监听是否显示菜单栏图标
		watchKey(globalStore.app, "showMenubarIcon", async (value) => {
			const tray = await getTrayById();

			tray?.setVisible(value);
		});

		// 监听语言变更
		watchKey(globalStore.appearance, "language", updateTrayMenu);
	});

	useUpdateEffect(() => {
		updateTrayMenu();

		emit(LISTEN_KEY.TOGGLE_LISTEN_CLIPBOARD, startListen);
	}, [startListen]);

	// 通过 id 获取托盘图标
	const getTrayById = () => {
		return TrayIcon.getById(TRAY_ID);
	};

	// 创建托盘图标
	const createTrayIcon = async () => {
		const tray = await getTrayById();

		if (tray) return;

		const { appName, appVersion } = globalStore.env;

		const menu = await getTrayMenu();

		const iconPath = isMac() ? "./assets/tray-mac.ico" : "./assets/tray.ico";
		const icon = await resolveResource(iconPath);

		const options: TrayIconOptions = {
			menu,
			icon,
			id: TRAY_ID,
			tooltip: `${appName} v${appVersion}`,
			iconAsTemplate: true,
			menuOnLeftClick: isMac(),
			action: (event) => {
				if (isMac()) return;

				if (event.type === "Click" && event.button === "Left") {
					showWindow("main");
				}
			},
		};

		return TrayIcon.new(options);
	};

	// 获取托盘菜单
	const getTrayMenu = async () => {
		const { appVersion } = globalStore.env;

		// TODO: 添加国际化
		const items = await Promise.all([
			MenuItem.new({
				text: "偏好设置",
				action: () => showWindow("preference"),
			}),
			MenuItem.new({
				text: startListen ? "停止监听" : "开始监听",
				action: toggle,
			}),
			PredefinedMenuItem.new({ item: "Separator" }),
			MenuItem.new({
				text: "关于",
				action: () => {
					showWindow();

					navigate("about");
				},
			}),
			MenuItem.new({
				text: "检查更新",
				action: () => {
					showWindow();

					emit(LISTEN_KEY.UPDATE_APP, true);
				},
			}),
			MenuItem.new({
				text: "开源地址",
				action: () => open(GITHUB_LINK),
			}),
			PredefinedMenuItem.new({ item: "Separator" }),
			MenuItem.new({
				text: `版本 ${appVersion}`,
				enabled: false,
			}),
			MenuItem.new({
				text: "退出",
				action: () => exit(1),
			}),
		]);

		return Menu.new({ items });
	};

	// 更新托盘菜单
	const updateTrayMenu = async () => {
		const tray = await getTrayById();

		if (!tray) return;

		const menu = await getTrayMenu();

		tray.setMenu(menu);
	};

	return <></>;
};

export default Tray;
