import Icon from "@/components/Icon";
import Update from "@/components/Update";
import MacosPermissions from "@/pages/General/components/MacosPermissions";
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
		if (isMac()) {
			frostedWindow();
		}

		if (!isWin()) {
			setTheme(globalStore.appearance.theme);
		}

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
			globalStore.appearance.language = payload;
		});

		watchKey(globalStore.app, "autoStart", async (value) => {
			const enabled = await isEnabled();

			if (value && !enabled) {
				return enable();
			}

			if (!value && enabled) {
				disable();
			}
		});

		watchKey(globalStore.appearance, "language", () => {
			requestAnimationFrame(() => {
				appWindow.setTitle(t("preference.title"));
			});
		});

		subscribeKey(globalStore.appearance, "theme", async (value) => {
			let nextTheme = value;

			if (isWin()) {
				const yes = await ask("切换主题需要重启 app 才能生效！", {
					okLabel: "重启",
					cancelLabel: "取消",
					type: "warning",
				});

				if (!yes) return;
			}

			if (nextTheme === "auto") {
				nextTheme = (await appWindow.theme()) ?? "light";
			}

			globalStore.appearance.isDark = nextTheme === "dark";

			setTheme(value);
		});
	});

	useRegister(toggleWindowVisible, [shortcut.preference]);

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

			<div
				data-tauri-drag-region
				className="h-full flex-1 overflow-auto bg-2 p-16 transition"
			>
				<Outlet />
			</div>

			<Update />

			<div hidden>
				<MacosPermissions />
			</div>
		</Flex>
	);
};

export default Preference;
