import Icon from "@/components/Icon";
import { getTauriVersion } from "@tauri-apps/api/app";
import { emit } from "@tauri-apps/api/event";
import { arch, version } from "@tauri-apps/api/os";
import { open } from "@tauri-apps/api/shell";
import { Button, Flex, Tooltip } from "antd";
import { useSnapshot } from "valtio";

const About = () => {
	const { appInfo, platform } = useSnapshot(globalStore);
	const { t } = useTranslation();

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
			<img src="/logo.png" className="h-120 w-120" alt="logo" />
			<Flex vertical align="center" gap="small">
				<div className="color-1 font-bold text-22 transition">
					{appInfo?.name}
				</div>
				<Flex align="center" gap={4}>
					<span>v{appInfo?.version}</span>
					<Tooltip title={t("preference.about.hints.update_tooltip")}>
						<Icon
							hoverable
							name="i-iconamoon:restart"
							size={16}
							onMouseDown={update}
						/>
					</Tooltip>
				</Flex>
				<span>{t("preference.about.hints.introduce")}</span>
				<Flex gap="middle">
					<Tooltip title={t("preference.about.hints.copy_tooltip")}>
						<Button size="large" type="primary" onClick={copyInfo}>
							{t("preference.about.button.copy_info")}
						</Button>
					</Tooltip>

					<Button
						ghost
						danger
						size="large"
						type="primary"
						onClick={feedbackIssue}
					>
						{t("preference.about.button.feedback_issue")}
					</Button>
				</Flex>
			</Flex>
		</Flex>
	);
};

export default About;
