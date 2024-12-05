import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { AlipayOutlined, QqOutlined, WechatOutlined } from "@ant-design/icons";
import { getTauriVersion } from "@tauri-apps/api/app";
import { emit } from "@tauri-apps/api/event";
import { arch, version } from "@tauri-apps/plugin-os";
import { Avatar, Button, Image, message } from "antd";
import { useSnapshot } from "valtio";
import Thank from "./components/Thank";

const About = () => {
	const { appearance, env } = useSnapshot(globalStore);
	const { t } = useTranslation();
	const [visible, { toggle }] = useBoolean();
	const [imageSrc, setImageSrc] = useState("");

	const theme = useCreation(() => {
		return appearance.isDark ? "dark" : "light";
	}, [appearance.isDark]);

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

		await writeText(JSON.stringify(info, null, 2));

		message.success(t("preference.about.about_software.hints.copy_success"));
	};

	const previewImage = (src: string) => {
		setImageSrc(WEBSITE_LINK + src);
		toggle();
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
						className="hover:b-wechat!"
						icon={<WechatOutlined className="text-wechat" />}
						onClick={() => {
							previewImage(`/community/wechat-group-${theme}.png`);
						}}
					/>
					<Button
						className="hover:b-qq!"
						icon={<QqOutlined className="text-qq" />}
						onClick={() => {
							previewImage(`/community/qq-group-${theme}.png`);
						}}
					/>
				</ProListItem>

				<ProListItem title={t("preference.about.about_software.label.sponsor")}>
					<Button
						className="hover:b-wechat!"
						icon={<WechatOutlined className="text-wechat" />}
						onClick={() => {
							previewImage("/sponsor/wechat-pay.png");
						}}
					/>
					<Button
						className="hover:b-alipay!"
						icon={<AlipayOutlined className="text-alipay" />}
						onClick={() => {
							previewImage("/sponsor/ali-pay.png");
						}}
					/>
				</ProListItem>

				<Image
					hidden
					preview={{
						visible,
						src: imageSrc,
						onVisibleChange: toggle,
					}}
				/>
			</ProList>

			<Thank />
		</>
	);
};

export default About;
