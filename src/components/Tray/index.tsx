import { emit } from "@tauri-apps/api/event";
import { Menu, MenuItem, PredefinedMenuItem } from "@tauri-apps/api/menu";
import { resolveResource } from "@tauri-apps/api/path";
import { TrayIcon, type TrayIconOptions } from "@tauri-apps/api/tray";
import { open } from "@tauri-apps/plugin-shell";

interface State {
	tray?: TrayIcon;
}

const Tray = () => {
	const navigate = useNavigate();
	const state = useReactive<State>({});
	const [startListen, { toggle }] = useBoolean(true);

	useMount(async () => {
		await createTrayIcon();

		// 监听是否显示菜单栏图标
		watchKey(globalStore.app, "showMenubarIcon", (value) => {
			state.tray?.setVisible(value);
		});

		// 监听语言变更
		watchKey(globalStore.appearance, "language", updateTrayMenu);
	});

	useUpdateEffect(() => {
		updateTrayMenu();

		emit(LISTEN_KEY.TOGGLE_LISTEN_CLIPBOARD, startListen);
	}, [startListen]);

	// 创建托盘图标
	const createTrayIcon = async () => {
		const { appName, appVersion } = globalStore.env;

		const menu = await getTrayMenu();

		const iconPath = isMac() ? "./assets/tray-mac.ico" : "./assets/tray.ico";
		const icon = await resolveResource(iconPath);

		const options: TrayIconOptions = {
			menu,
			icon,
			tooltip: `${appName} v${appVersion}`,
			iconAsTemplate: true,
			action: (event) => {
				if (isMac()) return;

				if (event.type === "Click" && event.button === "Left") {
					showWindow("main");
				}
			},
		};

		state.tray = await TrayIcon.new(options);
	};

	// 获取托盘菜单
	const getTrayMenu = async () => {
		const { appVersion } = globalStore.env;

		// TODO：添加国际化
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
			PredefinedMenuItem.new({ item: "Quit", text: "退出" }),
		]);

		return Menu.new({ items });
	};

	// 更新托盘菜单
	const updateTrayMenu = async () => {
		if (!state.tray) return;

		const menu = await getTrayMenu();

		state.tray.setMenu(menu);
	};

	return <></>;
};

export default Tray;
