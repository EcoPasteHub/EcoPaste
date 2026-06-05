import { theme as antdTheme } from "antd";
import type { Preset } from "unocss";

/**
 * UnoCSS preset：把 antd v6 颜色 token 以 `ant-*` 前缀接入 `theme.colors`。
 *
 * 只注册颜色 token：`color*` 语义色写作 `text-ant-secondary` /
 * `bg-ant-container`，调色板写作 `bg-ant-blue-1`。UnoCSS 只会为实际用到的
 * class 生成 CSS，所以注册完整颜色表不会增加最终样式体积。
 */
export interface PresetAntdColorsOptions {
  /** CSS 变量前缀，默认 `"ant"`，需与 antd `getPrefixCls()` 保持一致。 */
  antPrefix?: string;
}

type ColorMap = Record<string, string>;

const PALETTE_RE = /^([a-z]+)-(\d{1,2})$/;
const COLOR_PREFIX_ALIASES = ["bg-", "text-", "border-", "fill-"] as const;

/**
 * 转换 antd token key：`colorTextSecondary` -> `color-text-secondary`。
 */
const kebab = (value: string): string => {
  return value.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();
};

/**
 * 收集 antd 颜色 token，输出 UnoCSS `theme.colors` 可消费的平铺结构。
 */
const collectAntdColors = (antPrefix: string): ColorMap => {
  const colors: ColorMap = {};

  for (const key of Object.keys(antdTheme.getDesignToken())) {
    if (/^color[A-Z]/.test(key)) {
      const short = kebab(key.slice("color".length));
      const cssVariable = `var(--${antPrefix}-color-${short})`;
      colors[`ant-${short}`] = cssVariable;

      for (const prefix of COLOR_PREFIX_ALIASES) {
        if (!short.startsWith(prefix)) continue;

        colors[`ant-${short.slice(prefix.length)}`] ??= cssVariable;
      }
      continue;
    }

    const paletteMatch = PALETTE_RE.exec(key);
    if (!paletteMatch) continue;

    const [, palette, step] = paletteMatch;
    colors[`ant-${palette}-${step}`] = `var(--${antPrefix}-${key})`;
  }

  return colors;
};

/**
 * 创建 antd 颜色 preset。
 */
export const presetAntdColors = (
  options: PresetAntdColorsOptions = {},
): Preset => {
  const antPrefix = options.antPrefix ?? "ant";
  const colors = collectAntdColors(antPrefix);

  return {
    extendTheme: (theme: { colors?: ColorMap }) => {
      theme.colors = {
        ...theme.colors,
        ...colors,
      };
    },
    name: "preset-antd-colors",
  };
};

export default presetAntdColors;
