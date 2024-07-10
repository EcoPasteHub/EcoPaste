import { listen } from "@tauri-apps/api/event";
import { relaunch } from "@tauri-apps/api/process";
import {
	type UpdateManifest,
	installUpdate,
	onUpdaterEvent,
	checkUpdate as tauriCheckUpdate,
} from "@tauri-apps/api/updater";
import type { Timeout } from "ahooks/lib/useRequest/src/types";
import { Flex, Modal, Typography, message } from "antd";
import Markdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import { useSnapshot } from "valtio";

interface State {
	open?: boolean;
	loading?: boolean;
	manifest?: UpdateManifest;
}

const MESSAGE_KEY = "updatable";
let timer: Timeout;

const Update = () => {
	const { appInfo, autoUpdate } = useSnapshot(globalStore);

	const state = useReactive<State>({});

	const [messageApi, contextHolder] = message.useMessage();

	useMount(() => {
		listen(LISTEN_KEY.UPDATE, async () => {
			messageApi.open({
				key: MESSAGE_KEY,
				type: "loading",
				content: "正在检查更新...",
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

			if (shouldUpdate) {
				showWindow();

				messageApi.destroy(MESSAGE_KEY);

				Object.assign(state, { manifest, open: true });
			} else if (showMessage) {
				messageApi.open({
					key: MESSAGE_KEY,
					type: "success",
					content: "当前已是最新版本🎉",
				});
			}
		} catch {
			if (!showMessage) return;

			messageApi.open({
				key: MESSAGE_KEY,
				type: "error",
				content: "检查更新时出错，请检查网络并重试。",
			});
		}
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
						content: "正在下载最新的安装包...",
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
						content: "下载完成，即将开始安装并重启应用。",
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
				title="发现新版本🥳"
				okText="立即更新"
				cancelText="以后再说"
				confirmLoading={state.loading}
				onOk={handleOk}
				onCancel={handleCancel}
			>
				<Flex vertical gap="small" className="pt-4">
					<Flex align="center">
						更新版本：
						<span>
							v{appInfo?.version} 👉{" "}
							<span className="text-primary">v{state.manifest?.version}</span>
						</span>
					</Flex>

					<Flex align="center">
						更新时间：
						<span>{updateTime}</span>
					</Flex>

					<Flex vertical>
						更新日志：
						<Markdown
							className="revert-all max-h-220 overflow-auto"
							rehypePlugins={[rehypeRaw]}
							components={{
								a: (props) => {
									const { href, children } = props;

									return (
										<Typography.Link
											href={href}
											target="_blank"
											rel="noreferrer"
										>
											{children}
										</Typography.Link>
									);
								},
							}}
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
