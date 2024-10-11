import { listen } from "@tauri-apps/api/event";
import { relaunch } from "@tauri-apps/plugin-process";
import { type Update, check } from "@tauri-apps/plugin-updater";
import type { Timeout } from "ahooks/lib/useRequest/src/types";
import { Flex, Modal, Typography, message } from "antd";
import clsx from "clsx";
import Markdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import styles from "./index.module.scss";

const { Link, Text } = Typography;

interface State {
	open?: boolean;
	loading?: boolean;
	update?: Update;
}

let timer: Timeout;

const UpdateApp = () => {
	const { t } = useTranslation();
	const state = useReactive<State>({});
	const [messageApi, contextHolder] = message.useMessage();

	useMount(() => {
		// 监听更新事件
		listen<boolean>(LISTEN_KEY.UPDATE_APP, () => {
			checkUpdate(true);

			messageApi.open({
				key: UPDATE_MESSAGE_KEY,
				type: "loading",
				content: t("component.app_update.hints.checking_update"),
				duration: 0,
			});
		});

		// 监听自动更新配置变化
		watchKey(globalStore.update, "auto", (value) => {
			clearInterval(timer);

			if (!value) return;

			checkUpdate();

			timer = setInterval(checkUpdate, 1000 * 60 * 60 * 24);
		});

		// 监听参与测试版本配置变化
		watchKey(globalStore.update, "beta", () => checkUpdate());
	});

	// 检查更新
	const checkUpdate = async (showMessage = false) => {
		try {
			const update = await check({
				headers: {
					"join-beta": String(globalStore.update.beta),
				},
			});

			if (update?.available) {
				showWindow();

				const { version, currentVersion, body = "", date } = update;

				Object.assign(update, {
					version: `v${version}`,
					currentVersion: `v${currentVersion}`,
					body: replaceBody(body),
					date: dayjs.utc(date?.split(" ")?.slice(0, 2)?.join(" ")).local(),
				});

				Object.assign(state, { update, open: true });

				messageApi.destroy(UPDATE_MESSAGE_KEY);
			} else if (showMessage) {
				messageApi.open({
					key: UPDATE_MESSAGE_KEY,
					type: "success",
					content: t("component.app_update.hints.latest_version"),
				});
			}
		} catch (error: any) {
			if (!showMessage) return;

			messageApi.open({
				key: UPDATE_MESSAGE_KEY,
				type: "error",
				content: error,
			});
		}
	};

	// 替换更新日志里的内容
	const replaceBody = (body: string) => {
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

	const handleOk = () => {
		state.loading = true;

		state.update?.downloadAndInstall((progress) => {
			const { event } = progress;

			if (event !== "Finished") return;

			relaunch();
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
							{state.update?.currentVersion} 👉{" "}
							<Link
								href={`${GITHUB_LINK}/releases/tag/${state.update?.version}`}
							>
								{state.update?.version}
							</Link>
						</span>
					</Flex>

					<Flex align="center">
						{t("component.app_update.label.release_time")}：
						<span>
							{dayjs(state.update?.date).format("YYYY-MM-DD HH:mm:ss")}
						</span>
					</Flex>

					<Flex vertical>
						{t("component.app_update.label.release_notes")}：
						<Markdown
							className={clsx(styles.markdown, "max-h-220 overflow-auto")}
							rehypePlugins={[rehypeRaw]}
							components={{
								a: ({ href, children }) => <Link href={href}>{children}</Link>,
								mark: ({ children }) => <Text mark>{children}</Text>,
								code: ({ children }) => <Text code>{children}</Text>,
							}}
						>
							{state.update?.body}
						</Markdown>
					</Flex>
				</Flex>
			</Modal>
		</>
	);
};

export default UpdateApp;
