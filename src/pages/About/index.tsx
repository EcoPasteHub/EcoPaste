import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { WechatOutlined } from "@ant-design/icons";
import { emit } from "@tauri-apps/api/event";
import { arch, version } from "@tauri-apps/plugin-os";
import { Avatar, Button, Image, message } from "antd";
import { useSnapshot } from "valtio";
import Thank from "./components/Thank";

const About = () => {
	const { env } = useSnapshot(globalStore);
	const [visible, { toggle }] = useBoolean();
	const { t } = useTranslation();

	const copyInfo = async () => {
		const { appName, appVersion, platform } = env;

		const info = {
			appName,
			appVersion,
			platform,
			platformArch: await arch(),
			platformVersion: await version(),
		};

		await writeText(JSON.stringify(info, null, 2));

		message.success(t("preference.about.about_software.hints.copy_success"));
	};

	return (
		<>
			<ProList header={t("preference.about.about_software.title")}>
				<ProListItem
					avatar={<Avatar src="/logo.png" size={44} shape="square" />}
					title={env.appName}
					description={`${t("preference.about.about_software.label.version")}v${env.appVersion}`}
				>
					<Button
						type="primary"
						onClick={() => {
							emit(LISTEN_KEY.UPDATE_APP, true);
						}}
					>
						{t("preference.about.about_software.button.check_update")}
					</Button>
				</ProListItem>

				<ProListItem
					title={t("preference.about.about_software.label.software_info")}
					description={t("preference.about.about_software.hints.software_info")}
				>
					<Button onClick={copyInfo}>
						{t("preference.about.about_software.button.copy")}
					</Button>
				</ProListItem>

				<ProListItem
					title={t("preference.about.about_software.label.open_source_address")}
					description={<a href={GITHUB_LINK}>{GITHUB_LINK}</a>}
				>
					<Button danger href={GITHUB_ISSUES_LINK}>
						{t("preference.about.about_software.button.feedback_issue")}
					</Button>
				</ProListItem>

				<ProListItem
					title={t("preference.about.about_software.label.community")}
				>
					<Button
						className="hover:b-#2aae67!"
						icon={<WechatOutlined style={{ color: "#2aae67" }} />}
						onClick={toggle}
					/>
				</ProListItem>

				<Image
					hidden
					preview={{
						visible,
						src: "https://picture-bed.ayangweb.cn/EcoPaste/wechat.png",
						onVisibleChange: toggle,
					}}
				/>
			</ProList>

			<Thank />
		</>
	);
};

export default About;
