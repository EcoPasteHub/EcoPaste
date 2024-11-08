import isUrl from "is-url";

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
	return globalStore.env.platform === "macos";
};

/**
 * 是否为 windows 系统
 */
export const isWin = () => {
	return globalStore.env.platform === "windows";
};

/**
 * 是否为 linux 系统
 */
export const isLinux = () => {
	return globalStore.env.platform === "linux";
};

/**
 * 是否为链接
 */
export const isURL = (value: string) => {
	return isUrl(value);
};

/**
 * 是否为邮箱
 */
export const isEmail = (value: string) => {
	const regex = /^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$/;

	return regex.test(value);
};

/**
 * 是否为颜色
 */
export const isColor = (value: string) => {
	const excludes = [
		"none",
		"currentColor",
		"-moz-initial",
		"inherit",
		"initial",
		"revert",
		"revert-layer",
		"unset",
		"ActiveBorder",
		"ActiveCaption",
		"AppWorkspace",
		"Background",
		"ButtonFace",
		"ButtonHighlight",
		"ButtonShadow",
		"ButtonText",
		"CaptionText",
		"GrayText",
		"Highlight",
		"HighlightText",
		"InactiveBorder",
		"InactiveCaption",
		"InactiveCaptionText",
		"InfoBackground",
		"InfoText",
		"Menu",
		"MenuText",
		"Scrollbar",
		"ThreeDDarkShadow",
		"ThreeDFace",
		"ThreeDHighlight",
		"ThreeDLightShadow",
		"ThreeDShadow",
		"Window",
		"WindowFrame",
		"WindowText",
	];

	if (excludes.includes(value) || value.includes("url")) return false;

	const style = new Option().style;

	style.backgroundColor = value;
	style.backgroundImage = value;

	const { backgroundColor, backgroundImage } = style;

	return backgroundColor !== "" || backgroundImage !== "";
};

/**
 * 是否为图片
 */
export const isImage = (value: string) => {
	const regex = /\.(jpe?g|png|webp|avif|gif|svg|bmp|ico|tiff?|heic|apng)$/i;

	return regex.test(value);
};
