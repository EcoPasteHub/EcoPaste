/**
 * 返回一个延迟指定时间后解决的 Promise，用于异步操作中的延时控制。
 *
 * @param 等待的时间，单位为毫秒，默认为 1000 毫秒。
 */
export const wait = (ms = 1000) => {
	return new Promise((resolve) => setTimeout(resolve, ms));
};
