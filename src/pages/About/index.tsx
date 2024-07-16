import Icon from "@/components/Icon";
import { getTauriVersion } from "@tauri-apps/api/app";
import { emit } from "@tauri-apps/api/event";
import { arch, version } from "@tauri-apps/api/os";
import { open } from "@tauri-apps/api/shell";
import { Button, Flex, Tooltip } from "antd";
import { useSnapshot } from "valtio";

const About = () => {
	const { appInfo, platform } = useSnapshot(globalStore);

	const update = () => {
		emit(LISTEN_KEY.UPDATE);
	};

	const copyInfo = async () => {
		const info = {
			appName: appInfo?.name,
			appVersion: appInfo?.version,
			tauriVersion: await getTauriVersion(),
			platform,
			platformArch: await arch(),
			platformVersion: await version(),
		};

		writeText(JSON.stringify(info, null, 2));
	};

	const feedbackIssue = () => {
		open(`${GITHUB_ISSUES_LINK}/new/choose`);
	};

	return (
		<Flex
			vertical
			align="center"
			justify="center"
			className="color-2 h-full transition"
		>
			<img src="logo.png" className="h-120 w-120" alt="logo" />
			<Flex vertical align="center" gap="small">
				<div className="color-1 font-bold text-22 transition">
					{appInfo?.name}
				</div>
				<Flex align="center" gap={4}>
					<span>v{appInfo?.version}</span>
					<Tooltip title="检查更新">
						<Icon
							hoverable
							name="i-iconamoon:restart"
							size={16}
							onMouseDown={update}
						/>
					</Tooltip>
				</Flex>
				<span>开源的跨平台剪切板工具，让您的工作更加高效便捷。</span>
				<Flex gap="middle">
					<Tooltip title="复制应用和系统信息，用于 Issue">
						<Button size="large" type="primary" onClick={copyInfo}>
							复制信息
						</Button>
					</Tooltip>

					<Button
						ghost
						danger
						size="large"
						type="primary"
						onClick={feedbackIssue}
					>
						反馈问题
					</Button>
				</Flex>
			</Flex>
		</Flex>
	);
};

export default About;
