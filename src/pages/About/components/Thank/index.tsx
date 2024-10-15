import Icon from "@/components/Icon";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { useSnapshot } from "valtio";

const Thank = () => {
	const { env } = useSnapshot(globalStore);

	const list = [
		{
			name: "tauri",
			icon: "i-logos:tauri",
			link: "https://tauri.app",
		},
		{
			name: "tauri-plugins",
			icon: "i-skill-icons:tauri-dark",
			link: "https://github.com/tauri-apps/plugins-workspace",
		},
		{
			name: "clipboard-rs",
			icon: "i-skill-icons:rust",
			link: "https://github.com/ChurchTao/clipboard-rs",
		},
		{
			name: "ant-design",
			icon: "i-logos:ant-design",
			link: "https://ant-design.antgroup.com",
		},
		{
			name: "unocss",
			icon: "i-logos:unocss",
			link: "https://unocss.dev",
		},
		{
			name: "ahooks",
			icon: "i-skill-icons:javascript",
			link: "https://ahooks.js.org",
		},
		{
			name: "lodash-es",
			icon: "i-logos:lodash",
			link: "https://www.lodashjs.com",
		},
		{
			name: "i18next",
			icon: "i-simple-icons:i18next",
			link: "https://www.i18next.com",
			iconColor: "#26a69a",
		},
		{
			name: "valtio",
			icon: "i-skill-icons:v-dark",
			link: "https://github.com/pmndrs/valtio",
		},
	];

	return (
		<ProList header="致谢">
			{list.map((item) => {
				const { name, icon, link, iconColor } = item;

				return (
					<ProListItem
						key={name}
						title={name}
						avatar={<Icon name={icon} size={30} color={iconColor} />}
						description={<a href={link}>{link}</a>}
					/>
				);
			})}

			<ProListItem
				title={`感谢所有列出的和未能列出的开源依赖，以及为 ${env.appName} 提出建议的用户和做出贡献的开发者。`}
			/>
		</ProList>
	);
};

export default Thank;
