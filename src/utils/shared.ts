/**
 * 延迟执行
 * @param delay 延迟时间
 */
export const wait = (delay = 100) => {
	return new Promise((resolve) => setTimeout(resolve, delay));
};
