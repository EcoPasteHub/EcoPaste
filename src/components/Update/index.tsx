import { listen } from "@tauri-apps/api/event";
import { relaunch } from "@tauri-apps/api/process";
import {
	type UpdateManifest,
	installUpdate,
	onUpdaterEvent,
	checkUpdate as tauriCheckUpdate,
} from "@tauri-apps/api/updater";
import type { Timeout } from "ahooks/lib/useRequest/src/types";
import { Flex, Modal, message } from "antd";
import clsx from "clsx";
import Markdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import { useSnapshot } from "valtio";
import styles from "./index.module.scss";

interface State {
	open?: boolean;
	loading?: boolean;
	manifest?: UpdateManifest;
}

const MESSAGE_KEY = "updatable";
let timer: Timeout;

const Update = () => {
	const { appInfo, autoUpdate } = useSnapshot(globalStore);
	const { t } = useTranslation();

	const state = useReactive<State>({});

	const [messageApi, contextHolder] = message.useMessage();

	useMount(() => {
		listen(LISTEN_KEY.UPDATE, async () => {
			messageApi.open({
				key: MESSAGE_KEY,
				type: "loading",
				content: t("component.app_update.hints.checking_update"),
				duration: 0,
			});

			checkUpdate(true);
		});
	});

	useEffect(() => {
		clearInterval(timer);

		if (autoUpdate) {
			checkUpdate();

			timer = setInterval(checkUpdate, 1000 * 60 * 60 * 24);
		}
	}, [autoUpdate]);

	const updateTime = useCreation(() => {
		const date = state.manifest?.date?.split(" ")?.slice(0, 2)?.join(" ");

		return dayjs.utc(date).local().format("YYYY-MM-DD HH:mm:ss");
	}, [state.manifest?.date]);

	const checkUpdate = async (showMessage = false) => {
		try {
			const { shouldUpdate, manifest } = await tauriCheckUpdate();

			if (shouldUpdate && manifest) {
				showWindow();

				messageApi.destroy(MESSAGE_KEY);

				manifest.body = replaceManifestBody(manifest.body);

				Object.assign(state, { manifest, open: true });
			} else if (showMessage) {
				messageApi.open({
					key: MESSAGE_KEY,
					type: "success",
					content: t("component.app_update.hints.latest_version"),
				});
			}
		} catch {
			if (!showMessage) return;

			messageApi.open({
				key: MESSAGE_KEY,
				type: "error",
				content: t("component.app_update.hints.update_check_error"),
			});
		}
	};

	const replaceManifestBody = (body: string) => {
		return (
			body
				// 替换贡献者名称
				.replace(
					/(-.*?by.*?)@([^ ]+)/g,
					"$1<a href='https://github.com/$2'><mark>@$2</mark></a>",
				)
				// 替换 pr 链接
				.replace(
					new RegExp(`(${GITHUB_ISSUES_LINK}/)(\\d+)`, "g"),
					"[#$2]($1$2)",
				)
		);
	};

	const handleOk = async () => {
		state.loading = true;

		installUpdate();

		onUpdaterEvent((event) => {
			const { error, status } = event;

			switch (status) {
				case "DONE":
					relaunch();
					break;
				case "PENDING":
					messageApi.open({
						key: MESSAGE_KEY,
						type: "loading",
						content: t("component.app_update.hints.downloading_latest_package"),
						duration: 0,
					});
					break;
				case "ERROR":
					messageApi.open({
						key: MESSAGE_KEY,
						type: "error",
						content: error,
					});
					break;
				case "UPTODATE":
					messageApi.open({
						key: MESSAGE_KEY,
						type: "success",
						content: t("component.app_update.hints.download_complete_restart"),
					});
			}
		});
	};

	const handleCancel = () => {
		state.open = false;
	};

	return (
		<>
			{contextHolder}
			<Modal
				centered
				open={state.open}
				closable={false}
				keyboard={false}
				maskClosable={false}
				title={t("component.app_update.label.new_version_title")}
				okText={t("component.app_update.button.confirm_update")}
				cancelText={t("component.app_update.button.cancel_update")}
				className={styles.modal}
				confirmLoading={state.loading}
				onOk={handleOk}
				onCancel={handleCancel}
			>
				<Flex vertical gap="small" className="pt-4">
					<Flex align="center">
						{t("component.app_update.label.release_version")}：
						<span>
							v{appInfo?.version} 👉{" "}
							<a href={`${GITHUB_LINK}/releases/latest`}>
								v{state.manifest?.version}
							</a>
						</span>
					</Flex>

					<Flex align="center">
						{t("component.app_update.label.release_time")}：
						<span>{updateTime}</span>
					</Flex>

					<Flex vertical>
						{t("component.app_update.label.release_notes")}：
						<Markdown
							className={clsx(styles.markdown, "max-h-220 overflow-auto")}
							rehypePlugins={[rehypeRaw]}
						>
							{state.manifest?.body}
						</Markdown>
					</Flex>
				</Flex>
			</Modal>
		</>
	);
};

export default Update;
