import Icon from "@/components/Icon";
import ScrollRestore from "@/components/ScrollRestore";
import Tray from "@/components/Tray";
import UpdateApp from "@/components/UpdateApp";
import MacosPermissions from "@/pages/General/components/MacosPermissions";
import type { ClipboardItem } from "@/types/database";
import type { Store } from "@/types/store";
import { emit } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { disable, enable, isEnabled } from "@tauri-apps/plugin-autostart";
import { create, exists, readTextFile } from "@tauri-apps/plugin-fs";
import { Flex } from "antd";
import clsx from "clsx";
import { merge } from "lodash-es";
import { subscribe, useSnapshot } from "valtio";

const Preference = () => {
	const { pathname } = useLocation();
	const { shortcut } = useSnapshot(globalStore);
	const { t } = useTranslation();

	useMount(async () => {
		await restoreStore();

		const appWindow = getCurrentWebviewWindow();

		// 监听全局配置项变化
		subscribe(globalStore, handleStoreChanged);

		// 监听剪贴板配置项变化
		subscribe(clipboardStore, handleStoreChanged);

		// 监听自动启动变更
		watchKey(globalStore.app, "autoStart", async (value) => {
			const enabled = await isEnabled();

			if (value && !enabled) {
				return enable();
			}

			if (!value && enabled) {
				disable();
			}
		});

		// 监听语言变更
		watchKey(globalStore.appearance, "language", () => {
			requestAnimationFrame(() => {
				appWindow.setTitle(t("preference.title"));
			});
		});

		// 监听主题变更
		watchKey(globalStore.appearance, "theme", async (value) => {
			let nextTheme = value === "auto" ? null : value;

			await appWindow.setTheme(nextTheme);

			nextTheme = nextTheme ?? (await appWindow.theme());

			globalStore.appearance.isDark = nextTheme === "dark";
		});

		// 监听系统主题的变化
		appWindow.onThemeChanged(async () => {
			if (globalStore.appearance.theme !== "auto") return;

			globalStore.appearance.isDark = (await appWindow.theme()) === "dark";
		});
	});

	// 监听快捷键切换窗口显隐
	useRegister(toggleWindowVisible, [shortcut.preference]);

	// 每 30 分钟删除过期的历史数据
	useInterval(
		async () => {
			const { duration, unit } = clipboardStore.history;

			if (duration === 0) return;

			const list = await selectSQL<ClipboardItem[]>("history");

			for (const item of list) {
				const { createTime, favorite } = item;

				if (dayjs().diff(createTime, "days") >= duration * unit) {
					if (favorite) continue;

					deleteSQL("history", item);
				}
			}
		},
		1000 * 60 * 30,
	);

	// 配置项变化通知其它窗口和本地存储
	const handleStoreChanged = async () => {
		const store = { globalStore, clipboardStore };

		emit(LISTEN_KEY.STORE_CHANGED, { globalStore, clipboardStore });

		const file = await create(getSaveStorePath());
		await file.write(new TextEncoder().encode(JSON.stringify(store, null, 2)));
		await file.close();
	};

	// 从本地存储恢复配置项
	const restoreStore = async () => {
		const path = getSaveStorePath();

		const existed = await exists(path);

		if (!existed) return;

		const content = await readTextFile(path);
		const store: Store = JSON.parse(content);

		merge(globalStore, store.globalStore);
		merge(clipboardStore, store.clipboardStore);
	};

	return (
		<Flex className="h-screen">
			<Flex
				data-tauri-drag-region
				vertical
				gap="small"
				className={clsx("h-full w-200 p-12", [isMac() ? "pt-32" : "bg-1"])}
				onClick={(event) => event.stopPropagation()}
			>
				{preferenceRoute.children?.map((item) => {
					const { path, meta = {} } = item;
					const { title, icon } = meta;

					return (
						<Link
							key={title}
							to={path}
							className={clsx(
								"color-2! rounded-8 p-12 p-r-0 transition hover:bg-4",
								{
									"bg-primary! text-white!": pathname.endsWith(path),
								},
							)}
						>
							<Flex align="center" gap="small">
								<Icon name={icon} size={20} />
								<span className="font-bold">{t(title!)}</span>
							</Flex>
						</Link>
					);
				})}
			</Flex>

			<ScrollRestore
				data-tauri-drag-region
				className="h-full flex-1 overflow-auto bg-2 p-16 transition"
			>
				<Outlet />
			</ScrollRestore>

			<UpdateApp />

			<div hidden>
				<MacosPermissions />
			</div>

			<Tray />
		</Flex>
	);
};

export default Preference;
