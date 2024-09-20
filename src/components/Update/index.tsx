import { listen } from "@tauri-apps/api/event";
import { relaunch } from "@tauri-apps/api/process";
import {
	type UpdateManifest,
	installUpdate,
	onUpdaterEvent,
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

const MESSAGE_KEY = "update";

let timer: Timeout;

const Update = () => {
	const { env } = useSnapshot(globalStore);
	const { t } = useTranslation();
	const state = useReactive<State>({});
	const [messageApi, contextHolder] = message.useMessage();

	useMount(() => {
		// 监听更新事件
		listen<boolean>(LISTEN_KEY.UPDATE_APP, () => {
			check(true);

			messageApi.open({
				key: MESSAGE_KEY,
				type: "loading",
				content: t("component.app_update.hints.checking_update"),
				duration: 0,
			});
		});

		// 监听自动更新配置变化
		watchKey(globalStore.update, "auto", (value) => {
			clearInterval(timer);

			if (!value) return;

			check();

			timer = setInterval(check, 1000 * 60 * 60 * 24);
		});

		// 监听参与测试版本配置变化
		watchKey(globalStore.update, "beta", (value) => {
			if (!value) return;

			check();
		});
	});

	// 检查更新
	const check = async (showMessage = false) => {
		try {
			const { shouldUpdate, manifest } = await checkUpdate(
				globalStore.update.beta,
			);

			if (shouldUpdate && manifest) {
				const { version, body, date } = manifest;

				showWindow();

				messageApi.destroy(MESSAGE_KEY);

				Object.assign(manifest, {
					version: `v${version}`,
					body: replaceManifestBody(body),
					date: Number(date) * 1000,
				});

				Object.assign(state, { manifest, open: true });
			} else if (showMessage) {
				messageApi.open({
					key: MESSAGE_KEY,
					type: "success",
					content: t("component.app_update.hints.latest_version"),
				});
			}
		} catch (error: any) {
			state.loading = false;

			if (!showMessage) return;

			messageApi.open({
				key: MESSAGE_KEY,
				type: "error",
				content: error,
			});
		}
	};

	// 替换更新日志里的内容
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
					return relaunch();
				case "ERROR":
					state.loading = false;

					return messageApi.open({
						key: MESSAGE_KEY,
						type: "error",
						content: error,
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
				destroyOnClose
				open={state.open}
				closable={false}
				keyboard={false}
				maskClosable={false}
				title={t("component.app_update.label.new_version_title")}
				okText={t("component.app_update.button.confirm_update")}
				cancelText={t("component.app_update.button.cancel_update")}
				className={styles.modal}
				confirmLoading={state.loading}
				cancelButtonProps={{ disabled: state.loading }}
				onOk={handleOk}
				onCancel={handleCancel}
			>
				<Flex vertical gap="small" className="pt-4">
					<Flex align="center">
						{t("component.app_update.label.release_version")}：
						<span>
							v{env.appVersion} 👉{" "}
							<a
								href={`${GITHUB_LINK}/releases/tag/${state.manifest?.version}`}
							>
								{state.manifest?.version}
							</a>
						</span>
					</Flex>

					<Flex align="center">
						{t("component.app_update.label.release_time")}：
						<span>
							{dayjs(state.manifest?.date).format("YYYY-MM-DD HH:mm:ss")}
						</span>
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
