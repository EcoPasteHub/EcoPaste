import Icon from "@/components/Icon";
import ScrollRestore from "@/components/ScrollRestore";
import Update from "@/components/Update";
import MacosPermissions from "@/pages/General/components/MacosPermissions";
import type { ClipboardItem } from "@/types/database";
import type { Language } from "@/types/store";
import { emit, listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/api/shell";
import { appWindow } from "@tauri-apps/api/window";
import { Flex } from "antd";
import clsx from "clsx";
import { disable, enable, isEnabled } from "tauri-plugin-autostart-api";
import { subscribe, useSnapshot } from "valtio";
import { subscribeKey } from "valtio/utils";

const Preference = () => {
	const { pathname } = useLocation();
	const navigate = useNavigate();
	const { shortcut } = useSnapshot(globalStore);
	const { t } = useTranslation();

	useMount(async () => {
		// 监听全局状态变化
		subscribe(globalStore, () => {
			emit(LISTEN_KEY.GLOBAL_STORE_CHANGED, globalStore);
		});

		// 监听剪切板状态变化
		subscribe(clipboardStore, () => {
			emit(LISTEN_KEY.CLIPBOARD_STORE_CHANGED, clipboardStore);
		});

		// 监听打开 github 地址
		listen(LISTEN_KEY.GITHUB, () => {
			open(GITHUB_LINK);
		});

		// 监听打开关于页面
		listen(LISTEN_KEY.ABOUT, () => {
			showWindow();

			navigate("about");
		});

		// 监听语言变更
		listen<Language>(LISTEN_KEY.CHANGE_LANGUAGE, ({ payload }) => {
			globalStore.appearance.language = payload;
		});

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
			setLocale();

			requestAnimationFrame(() => {
				appWindow.setTitle(t("preference.title"));
			});
		});

		// 监听主题变更
		subscribeKey(globalStore.appearance, "theme", async (value) => {
			let nextTheme = value;

			if (nextTheme === "auto") {
				nextTheme = (await appWindow.theme()) ?? "light";
			}

			globalStore.appearance.isDark = nextTheme === "dark";

			setTheme(value);
		});

		// 监听是否显示菜单栏图标
		watchKey(globalStore.app, "showMenubarIcon", setTrayVisible);
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
				const { id, createTime, favorite } = item;

				if (dayjs().diff(createTime, "days") >= duration * unit) {
					if (favorite) continue;

					deleteSQL("history", id);
				}
			}
		},
		1000 * 60 * 30,
	);

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

			<Update />

			<div hidden>
				<MacosPermissions />
			</div>
		</Flex>
	);
};

export default Preference;
