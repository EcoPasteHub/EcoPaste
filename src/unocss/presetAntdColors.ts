import { theme as antdTheme } from "antd";
import type { Preset, Rule } from "unocss";

/**
 * UnoCSS preset：把 antd v6 颜色 token 暴露成原子类。
 * 参考 https://github.com/antdv-next/css-plugin/tree/main/packages/unocss，
 * 只保留颜色相关规则（color / background-color / border-color）。
 *
 * **不硬编码 token 名**：构建期从 `antd.theme.getDesignToken()` 枚举所有
 * `color*` 键与 `{palette}-{1..10}` 调色板键，按 antd CSS 变量命名规则
 * （camelCase → kebab-case，前缀 `--ant-`）映射，升级 antd 版本时自动跟进。
 *
 * 支持的类名（{c} = token 短名，例如 `primary` / `primary-hover` / `bg-container`
 * / `text-secondary` / `border-secondary` / `split` / `blue` / `blue-6`）：
 *   - color-{c} / c-{c}                 → color
 *   - bg-{c}                            → background-color
 *   - border-{c} / b-{c}                → border-color（四边）
 *   - border-{t|r|b|l|x|y}-{c}
 *     b-{t|r|b|l|x|y}-{c}
 *     b{t|r|b|l|x|y}-{c}                → 单/双边 border-*-color
 *
 * 形如 `border` / `text` / `fill` 这类无后缀的根 token 额外注册 `-default` 别名，
 * 所以 `b-border-default` 与 `b-border` 等价。
 */
export interface PresetAntdColorsOptions {
  /** CSS 变量前缀，默认 "ant"（对应 ConfigProvider 的 cssVar.prefix）。 */
  antPrefix?: string;
}

/** 边方向 → CSS 属性名映射。空字符串代表四边。 */
const SIDE_PROPS: Record<string, string[]> = {
  "": ["border-color"],
  b: ["border-bottom-color"],
  l: ["border-left-color"],
  r: ["border-right-color"],
  t: ["border-top-color"],
  x: ["border-left-color", "border-right-color"],
  y: ["border-top-color", "border-bottom-color"],
};

/** camelCase → kebab-case，匹配 antd cssinjs 生成 CSS 变量名的规则。 */
const kebab = (s: string): string =>
  s.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();

/**
 * 构建 token 短名 → CSS var 表达式查找表。
 * 从 antd 设计 token 中提取：
 *   - `color*` → 短名是去掉 `color` 前缀后 kebab-case（`colorPrimaryHover` →
 *     `primary-hover`），CSS var 是 `--{antPrefix}-color-{shortName}`。
 *   - `{palette}-{n}`（n = 1..10）→ 短名直接复用键名，CSS var 是
 *     `--{antPrefix}-{palette}-{n}`；同时注册 bare `{palette}` 指向 `-6`。
 */
const buildColorMap = (antPrefix: string): Map<string, string> => {
  const tokens = antdTheme.getDesignToken();
  const map = new Map<string, string>();
  const roots = new Set<string>();
  const palettes = new Set<string>();

  for (const key of Object.keys(tokens)) {
    if (/^color[A-Z]/.test(key)) {
      const shortName = kebab(key.slice("color".length));
      map.set(shortName, `var(--${antPrefix}-color-${shortName})`);
      const dash = shortName.indexOf("-");
      roots.add(dash > 0 ? shortName.slice(0, dash) : shortName);
      continue;
    }
    const paletteMatch = /^([a-z]+)-(\d{1,2})$/.exec(key);
    if (paletteMatch) {
      const [, palette, n] = paletteMatch;
      const level = Number(n);
      if (level >= 1 && level <= 10) {
        map.set(
          `${palette}-${level}`,
          `var(--${antPrefix}-${palette}-${level})`,
        );
        palettes.add(palette);
      }
    }
  }

  for (const root of roots) {
    if (map.has(root) && !map.has(`${root}-default`)) {
      map.set(`${root}-default`, map.get(root)!);
    }
  }
  for (const palette of palettes) {
    if (!map.has(palette)) {
      map.set(palette, `var(--${antPrefix}-${palette}-6)`);
    }
  }
  return map;
};

export const presetAntdColors = (
  options: PresetAntdColorsOptions = {},
): Preset => {
  const antPrefix = options.antPrefix ?? "ant";
  const colors = buildColorMap(antPrefix);
  const lookup = (name: string): string | undefined => colors.get(name);

  const rules: Rule[] = [
    [
      /^c(?:olor)?-(.+)$/,
      ([, name]) => {
        const v = lookup(name);
        return v ? { color: v } : undefined;
      },
    ],
    [
      /^bg-(.+)$/,
      ([, name]) => {
        const v = lookup(name);
        return v ? { "background-color": v } : undefined;
      },
    ],
    [
      /^(?:border|b)-(.+)$/,
      ([, name]) => {
        const v = lookup(name);
        return v ? { "border-color": v } : undefined;
      },
    ],
    [
      /^(?:border|b)-([trblxy])-(.+)$/,
      ([, side, name]) => {
        const v = lookup(name);
        if (!v) return undefined;
        return Object.fromEntries(SIDE_PROPS[side].map((k) => [k, v]));
      },
    ],
    [
      /^b([trblxy])-(.+)$/,
      ([, side, name]) => {
        const v = lookup(name);
        if (!v) return undefined;
        return Object.fromEntries(SIDE_PROPS[side].map((k) => [k, v]));
      },
    ],
  ];

  return {
    name: "preset-antd-colors",
    rules,
  };
};

export default presetAntdColors;
