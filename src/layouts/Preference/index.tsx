import Icon from "@/components/Icon";
import Update from "@/components/Update";
import type { Language } from "@/types/store";
import { emit, listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/api/shell";
import { appWindow } from "@tauri-apps/api/window";
import { Flex } from "antd";
import clsx from "clsx";
import { disable, enable, isEnabled } from "tauri-plugin-autostart-api";
import { subscribe, useSnapshot } from "valtio";

const Preference = () => {
	const { pathname } = useLocation();
	const navigate = useNavigate();
	const { wakeUpKey, autoStart, language } = useSnapshot(globalStore);
	const { theme, isDark, toggleTheme } = useTheme();
	const { t } = useTranslation();

	useMount(async () => {
		const autoLaunched = await isAutoLaunch();

		if (!autoLaunched) {
			showWindow();
		}

		createWindow("/");

		if (!isWin()) {
			toggleTheme(theme);
		}

		navigate("clipboard");

		subscribe(globalStore, () => {
			emit(LISTEN_KEY.GLOBAL_STORE_CHANGED, globalStore);
		});

		subscribe(clipboardStore, () => {
			emit(LISTEN_KEY.CLIPBOARD_STORE_CHANGED, clipboardStore);
		});

		listen(LISTEN_KEY.GITHUB, () => {
			open(GITHUB_LINK);
		});

		listen(LISTEN_KEY.ABOUT, () => {
			showWindow();

			navigate("about");
		});

		listen(LISTEN_KEY.TRAY_CLICK, () => {
			if (isMac() || globalStore.trayClick === "none") return;

			showWindow();
		});

		listen<Language>(LISTEN_KEY.CHANGE_LANGUAGE, ({ payload }) => {
			globalStore.language = payload;
		});
	});

	useAsyncEffect(async () => {
		const enabled = await isEnabled();

		if (autoStart && !enabled) {
			return enable();
		}

		if (!autoStart && enabled) {
			disable();
		}
	}, [autoStart]);

	useRegister(toggleWindowVisible, [wakeUpKey]);

	useEffect(() => {
		requestAnimationFrame(() => {
			appWindow.setTitle(t("preference.title"));
		});
	}, [language]);

	return (
		<Flex className="h-screen">
			<Flex
				data-tauri-drag-region
				vertical
				align="center"
				justify="space-between"
				className={clsx(
					"color-2 h-full w-90 bg-2 px-12 pb-32 text-center transition",
					[isWin() ? "pt-32" : "pt-48"],
				)}
			>
				<Flex vertical gap="large" onClick={(event) => event.stopPropagation()}>
					{preferenceRoute.children?.map((item) => {
						const { path, meta = {} } = item;
						const { title, icon } = meta;

						return (
							<Link
								key={title}
								to={path}
								className={clsx("hover:text-primary", {
									"text-primary": pathname.endsWith(path),
								})}
							>
								<Flex vertical align="center" gap={4}>
									<Icon name={icon} size={22} />
									<span>{t(title!)}</span>
								</Flex>
							</Link>
						);
					})}
				</Flex>

				<Icon
					hoverable
					size={24}
					name={isDark ? "i-iconamoon:mode-light" : "i-iconamoon:mode-dark"}
					onMouseDown={() => toggleTheme()}
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

export default Preference;
