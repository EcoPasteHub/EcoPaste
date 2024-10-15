import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { WechatOutlined } from "@ant-design/icons";
import { emit } from "@tauri-apps/api/event";
import { arch, version } from "@tauri-apps/plugin-os";
import { Avatar, Button, Image, message } from "antd";
import { useSnapshot } from "valtio";
import Thank from "./components/Thank";

// TODO: 国际化
const About = () => {
	const { env } = useSnapshot(globalStore);
	const [visible, { toggle }] = useBoolean();

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

		message.success("复制成功");
	};

	return (
		<>
			<ProList header="关于软件">
				<ProListItem
					avatar={<Avatar src="/logo.png" size={44} shape="square" />}
					title={env.appName}
					description={`版本：v${env.appVersion}`}
				>
					<Button
						type="primary"
						onClick={() => {
							emit(LISTEN_KEY.UPDATE_APP, true);
						}}
					>
						检查更新
					</Button>
				</ProListItem>

				<ProListItem
					title="软件信息"
					description="复制软件信息并提供给 Bug Issue"
				>
					<Button onClick={copyInfo}>复制</Button>
				</ProListItem>

				<ProListItem
					title="开源地址"
					description={<a href={GITHUB_LINK}>{GITHUB_LINK}</a>}
				>
					<Button danger href={GITHUB_ISSUES_LINK}>
						反馈问题
					</Button>
				</ProListItem>

				<ProListItem title="社区交流">
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
