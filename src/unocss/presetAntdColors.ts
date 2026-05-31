import { theme as antdTheme, type GlobalToken, type ThemeConfig } from "antd";
import type { CSSObject, Preset } from "unocss";

/**
 * UnoCSS preset：把 antd v6 颜色 token 暴露成原子类。
 *
 * 构建期从 `antd.theme.getDesignToken()` 枚举所有 `color*` 与
 * `{palette}-{1..10}` 调色板键，按 antd CSS 变量命名规则（camelCase →
 * kebab-case，前缀 `--ant-`）映射为 `var(--ant-...)`；同时通过 `preflights`
 * 把 token 写入 `:root`（不依赖 `ConfigProvider.cssVar`，因为后者只把
 * 变量挂在组件级 `xxx-css-var` 类名下，原生 DOM 元素拿不到）。
 *
 * 支持多主题：传 `themes` 时按选择器输出多套变量，切主题只需切类名。
 *
 * 类名形态（{c} 例：`primary` / `primary-hover` / `bg-container`
 * / `text-secondary` / `split` / `blue` / `blue-6`）：
 *   text-{c} / c-{c} / color-{c}        → color
 *   bg-{c}                              → background-color
 *   border-{c} / b-{c}                  → border-color
 *   border-{t|r|b|l|x|y}-{c}
 *     / b-{t|r|b|l|x|y}-{c}
 *     / b{t|r|b|l|x|y}-{c}              → 单/双边 border-*-color
 *
 * lookup 不命中时返回 `undefined`，让 wind4 等其它 preset 接管。
 */
export interface PresetAntdColorsOptions {
  /** CSS 变量前缀，默认 `"ant"`。 */
  antPrefix?: string;
  /**
   * 多主题输出。默认输出 light（`:root, .light`）与 dark（`.dark`）两套变量，
   * 切主题只需在 `<html>` 上切 `light` / `dark` 类。
   */
  themes?: Record<string, ThemeConfig>;
}

const SIDE_PROPS: Record<string, string[]> = {
  "": ["border-color"],
  b: ["border-bottom-color"],
  l: ["border-left-color"],
  r: ["border-right-color"],
  t: ["border-top-color"],
  x: ["border-left-color", "border-right-color"],
  y: ["border-top-color", "border-bottom-color"],
};

const kebab = (s: string): string =>
  s.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();

/** token 键名 → CSS 变量名。 */
const toVarName = (key: string, antPrefix: string): string => {
  if (/^color[A-Z]/.test(key)) {
    return `--${antPrefix}-color-${kebab(key.slice("color".length))}`;
  }
  return `--${antPrefix}-${/^[a-z]+-\d{1,2}$/.test(key) ? key : kebab(key)}`;
};

/** 颜色查找表：短名 → `var(--ant-...)`。 */
const buildColorMap = (
  token: GlobalToken,
  antPrefix: string,
): Map<string, string> => {
  const map = new Map<string, string>();
  const palettes = new Set<string>();
  for (const key of Object.keys(token)) {
    if (/^color[A-Z]/.test(key)) {
      const short = kebab(key.slice("color".length));
      map.set(short, `var(--${antPrefix}-color-${short})`);
    } else if (/^[a-z]+-\d{1,2}$/.test(key)) {
      map.set(key, `var(--${antPrefix}-${key})`);
      palettes.add(key.split("-")[0]);
    }
  }
  for (const p of palettes) map.set(p, `var(--${antPrefix}-${p}-6)`);
  return map;
};

/**
 * 创建 antd 颜色原子类 preset。
 */
export const presetAntdColors = (
  options: PresetAntdColorsOptions = {},
): Preset => {
  const antPrefix = options.antPrefix ?? "ant";
  const themes = options.themes ?? {
    ":root, .light": { algorithm: antdTheme.defaultAlgorithm },
    ".dark": { algorithm: antdTheme.darkAlgorithm },
  };
  const colors = buildColorMap(
    antdTheme.getDesignToken(Object.values(themes)[0]),
    antPrefix,
  );

  const sideProps = (side: string, v: string): CSSObject =>
    Object.fromEntries(SIDE_PROPS[side].map((p) => [p, v]));

  const preflightCss = Object.entries(themes)
    .map(([selector, cfg]) => {
      const lines = Object.entries(antdTheme.getDesignToken(cfg))
        .filter(([, v]) => typeof v === "string" || typeof v === "number")
        .map(([k, v]) => `  ${toVarName(k, antPrefix)}: ${v};`);
      return `${selector} {\n${lines.join("\n")}\n}`;
    })
    .join("\n");

  return {
    name: "preset-antd-colors",
    preflights: [{ getCSS: () => preflightCss }],
    rules: [
      [
        /^(?:text|c|color)-(.+)$/,
        ([, name]) => {
          const v = colors.get(name);
          return v ? { color: v } : undefined;
        },
      ],
      [
        /^bg-(.+)$/,
        ([, name]) => {
          const v = colors.get(name);
          return v ? { "background-color": v } : undefined;
        },
      ],
      [
        /^(?:border|b)(?:-([trblxy]))?-(.+)$/,
        ([, side, name]) => {
          const v = colors.get(name);
          return v ? sideProps(side ?? "", v) : undefined;
        },
      ],
      [
        /^b([trblxy])-(.+)$/,
        ([, side, name]) => {
          const v = colors.get(name);
          return v ? sideProps(side, v) : undefined;
        },
      ],
    ],
  };
};

export default presetAntdColors;
