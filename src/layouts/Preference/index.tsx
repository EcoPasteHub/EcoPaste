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
	const { theme, toggleTheme } = useTheme();
	const { t } = useTranslation();

	useMount(async () => {
		if (isMac()) {
			frostedWindow();
		}

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
							className={clsx("color-2! rounded-8 p-12 transition hover:bg-4", {
								"bg-primary! text-white!": pathname.endsWith(path),
							})}
						>
							<Flex align="center" gap="small">
								<Icon name={icon} size={20} />
								<span className="font-bold">{t(title!)}</span>
							</Flex>
						</Link>
					);
				})}
			</Flex>

			<div
				data-tauri-drag-region
				className="h-full flex-1 overflow-auto bg-2 p-16 transition"
			>
				<Outlet />
			</div>

			<Update />
		</Flex>
	);
};

export default Preference;
