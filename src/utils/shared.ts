/**
 * 延迟执行
 * @param delay 延迟时间
 */
export const wait = (delay = 100) => {
	return new Promise((resolve) => setTimeout(resolve, delay));
};

/**
 * requestAnimationFrame 循环，只在最后一次执行回调
 * @param fn 回调函数
 * @param count 执行次数
 */
export const raf = (fn: () => void, count = 1) => {
	if (count <= 1) {
		return requestAnimationFrame(fn);
	}

	requestAnimationFrame(() => {
		raf(fn, count - 1);
	});
};
