import { isArray, mergeWith } from "lodash-es";

/**
 * 深度递归合并两个对象，普通对象会递归合并，其他值会直接覆盖
 * @param target 目标对象
 * @param source 源对象
 */
export const deepAssign = <T, S>(target: T, source: S): T & S => {
	return mergeWith(target, source, (targetValue, sourceValue) => {
		if (isArray(targetValue)) {
			return sourceValue;
		}
	});
};
