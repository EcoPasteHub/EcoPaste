import { type } from "@tauri-apps/api/os";

/**
 * 是否为开发环境
 */
export const isDev = () => {
	return import.meta.env.DEV;
};

/**
 * 是否为 windows 系统
 */
export const isWin = async () => {
	const osType = await type();

	return osType === "Windows_NT";
};

/**
 * 是否为 mac 系统
 */
export const isMac = async () => {
	const osType = await type();

	return osType === "Darwin";
};

/**
 * 是否为链接
 */
export const isURL = (value: string) => {
	const regexp =
		/^(((ht|f)tps?):\/\/)?([^!@#$%^&*?.\s-]([^!@#$%^&*?.\s]{0,63}[^!@#$%^&*?.\s])?\.)+[a-z]{2,6}\/?/;

	return regexp.test(value);
};

/**
 * 是否为邮箱
 */
export const isEmail = (value: string) => {
	const regexp =
		/^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$/;

	return regexp.test(value);
};

/**
 * 是否为颜色
 */
export const isColor = (value: string) => {
	const style = new Option().style;

	style.color = value;
	style.backgroundImage = value;

	const { color, backgroundImage } = style;

	return color !== "" || backgroundImage !== "";
};
