import type { Language } from "@/types/store";
import enUS from "antd/locale/en_US";
import zhCN from "antd/locale/zh_CN";

const languageMap: Record<Language, typeof enUS> = {
	"zh-CN": zhCN,
	"en-US": enUS,
};

export const getAntdLocale = (language?: Language) => {
	return language && languageMap[language];
};
