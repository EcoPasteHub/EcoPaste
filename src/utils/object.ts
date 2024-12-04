import { isEqual, isPlainObject } from "lodash-es";

/**
 * 递归合并两个对象，普通对象会递归合并，其他对象和值会被直接分配覆盖
 * @param target 目标对象
 * @param source 源对象
 */
export const merge = (
	target: Record<string, any>,
	source: Record<string, any>,
) => {
	for (const key in source) {
		if (isPlainObject(source[key])) {
			merge(target[key], source[key]);
		} else {
			if (isEqual(target[key], source[key])) continue;

			target[key] = source[key];
		}
	}
};
