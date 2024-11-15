import { emit } from "@tauri-apps/api/event";
import { Menu, MenuItem, PredefinedMenuItem } from "@tauri-apps/api/menu";
import { resolveResource } from "@tauri-apps/api/path";
import { TrayIcon, type TrayIconOptions } from "@tauri-apps/api/tray";
import { exit, relaunch } from "@tauri-apps/plugin-process";
import { open } from "@tauri-apps/plugin-shell";

const Tray = () => {
	const [startListen, { toggle }] = useBoolean(true);
	const { t } = useTranslation();

	useMount(async () => {
		await createTrayIcon();

		// 监听是否显示菜单栏图标
		subscribeKey(
			globalStore.app,
			"showMenubarIcon",
			async (value) => {
				const tray = await getTrayById();

				tray?.setVisible(value);
			},
			true,
		);

		// 监听语言变更
		subscribeKey(globalStore.appearance, "language", updateTrayMenu, true);
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

		const iconPath = isMac() ? "assets/tray-mac.ico" : "assets/tray.ico";
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

		const items = await Promise.all([
			MenuItem.new({
				text: t("component.tray.label.preference"),
				action: () => showWindow("preference"),
			}),
			MenuItem.new({
				text: startListen
					? t("component.tray.label.stop_listening")
					: t("component.tray.label.start_listening"),
				action: toggle,
			}),
			PredefinedMenuItem.new({ item: "Separator" }),
			MenuItem.new({
				text: t("component.tray.label.check_update"),
				action: () => {
					showWindow();

					emit(LISTEN_KEY.UPDATE_APP, true);
				},
			}),
			MenuItem.new({
				text: t("component.tray.label.open_source_address"),
				action: () => open(GITHUB_LINK),
			}),
			PredefinedMenuItem.new({ item: "Separator" }),
			MenuItem.new({
				text: `${t("component.tray.label.version")} ${appVersion}`,
				enabled: false,
			}),
			MenuItem.new({
				text: t("component.tray.label.relaunch"),
				action: relaunch,
			}),
			MenuItem.new({
				text: t("component.tray.label.exit"),
				action: () => exit(0),
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
