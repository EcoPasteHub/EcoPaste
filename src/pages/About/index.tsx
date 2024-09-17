import Icon from "@/components/Icon";
import { getTauriVersion } from "@tauri-apps/api/app";
import { emit } from "@tauri-apps/api/event";
import { arch, version } from "@tauri-apps/api/os";
import { open } from "@tauri-apps/api/shell";
import { Button, Flex, Tooltip } from "antd";
import { useSnapshot } from "valtio";

const About = () => {
	const { env } = useSnapshot(globalStore);
	const { t } = useTranslation();

	const copyInfo = async () => {
		const { appName, appVersion, platform } = env;

		const info = {
			appName,
			appVersion,
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
			data-tauri-drag-region
			vertical
			align="center"
			justify="center"
			className="color-2 h-full transition"
		>
			<img src="/logo.png" className="h-100 h-100" alt="logo" />

			<Flex vertical align="center" gap="small">
				<div className="color-1 font-bold text-20 transition">
					{env.appName}
				</div>

				<Flex align="center" gap={4}>
					<span>v{env.appVersion}</span>
					<Tooltip title={t("preference.about.hints.update_tooltip")}>
						<Icon
							hoverable
							name="i-iconamoon:restart"
							size={16}
							onMouseDown={() => {
								emit(LISTEN_KEY.UPDATE_APP, true);
							}}
						/>
					</Tooltip>
				</Flex>

				<span className="text-center">
					{t("preference.about.hints.introduce")}
				</span>

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
