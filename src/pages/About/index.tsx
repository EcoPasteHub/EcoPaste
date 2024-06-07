import logo from "@/assets/img/logo.png";
import Icon from "@/components/Icon";
import { emit } from "@tauri-apps/api/event";
import { Button, Flex, Tooltip } from "antd";
import { useSnapshot } from "valtio";

const About = () => {
	const { appInfo } = useSnapshot(globalStore);

	return (
		<Flex
			vertical
			align="center"
			justify="center"
			className="color-2 h-full transition"
		>
			<img src={logo} className="h-120 w-120" alt="logo" />
			<Flex vertical align="center" gap="small">
				<div className="color-1 font-bold text-22 transition">
					{appInfo?.name}
				</div>
				<Flex align="center" gap={4}>
					<span>v{appInfo?.version}</span>
					<Tooltip title="检查更新">
						<Icon hoverable name="i-iconamoon:restart" size={16} />
					</Tooltip>
				</Flex>
				<span>开源的跨平台剪切板工具，让您的工作更加高效便捷。</span>
				<Button
					ghost
					size="large"
					type="primary"
					onClick={() => emit("github")}
				>
					给 {appInfo?.name} 点个 Star 吧👍
				</Button>
			</Flex>
		</Flex>
	);
};

export default About;
