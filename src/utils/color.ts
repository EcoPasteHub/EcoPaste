import { theme } from "antd";
import { kebabCase } from "es-toolkit";
import { map } from "es-toolkit/compat";

const { getDesignToken, darkAlgorithm } = theme;

/**
 * 生成 antd 的颜色变量
 */
export const generateColorVars = () => {
	const colors = [
		getDesignToken(),
		getDesignToken({ algorithm: darkAlgorithm }),
	];

	for (const [index, item] of colors.entries()) {
		const isDark = index !== 0;

		const vars: Record<string, any> = {};

		for (const [key, value] of Object.entries(item)) {
			vars[`--ant-${kebabCase(key)}`] = value;
		}

		const style = document.createElement("style");

		style.dataset.theme = isDark ? "dark" : "light";

		const selector = isDark ? "html.dark" : ":root";

		const values = map(vars, (value, key) => `${key}: ${value};`);

		style.innerHTML = `${selector}{\n${values.join("\n")}\n}`;

		document.head.appendChild(style);
	}
};
