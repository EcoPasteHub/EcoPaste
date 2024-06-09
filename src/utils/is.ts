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
