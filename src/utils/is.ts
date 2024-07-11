/**
 * 是否为开发环境
 */
export const isDev = () => {
	return import.meta.env.DEV;
};

/**
 * 是否为 macos 系统
 */
export const isMac = () => {
	return globalStore.platform === "Darwin";
};

/**
 * 是否为 windows 系统
 */
export const isWin = () => {
	return globalStore.platform === "Windows_NT";
};

/**
 * 是否为 linux 系统
 */
export const isLinux = () => {
	return globalStore.platform === "Linux";
};

/**
 * 是否为链接
 */
export const isURL = (value: string) => {
	const regexp =
		/^(https?:\/\/)?((localhost)|(([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}))(:\d+)?(\/[a-zA-Z0-9\-._~:\/?#@!$&'()*+,;=%]*)?$/;

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

	style.background = value;

	const { background } = style;

	return background !== "";
};
