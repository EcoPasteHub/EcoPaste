import { theme as antdTheme, type ThemeConfig } from "antd";
import type { Preset } from "unocss";

/**
 * UnoCSS preset：把 antd v6 颜色 token 接入 UnoCSS 的 `theme.colors`。
 *
 * 做两件事：
 *   1. `extendTheme`：把所有 `color*` 与 `{palette}-{1..10}` 写入
 *      `theme.colors`（值为 `var(--ant-...)`）。wind4 的 `parseColor`
 *      检测到 `var(` 时会原样保留色值，所以 wind4 自带的 **所有** 用色工具
 *      （text- / bg- / from- / via- / to- / border- / ring- / shadow-
 *      / outline- / decoration- / accent- / fill- / stroke- ...）
 *      都会自动支持 antd 颜色，无需我们再单独写规则。
 *   2. `preflights`：把 antd token 落到 `:root` / `.dark`，供上面那些
 *      `var(--ant-...)` 在运行时取到值；切主题只需在 `<html>` 切类名。
 *
 * 调色板键以嵌套对象（`blue: { "1": ..., DEFAULT: ... }`）写入，避免
 * 整体覆盖 wind4 自带的 Tailwind 调色板（`blue-50`..`blue-950` 仍可用）。
 * 单调色板别名（`text-blue`）通过 `DEFAULT` 指向 `var(--ant-blue-6)`。
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

const kebab = (s: string): string =>
  s.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();

const PALETTE_RE = /^([a-z]+)-(\d{1,2})$/;

/**
 * 创建 antd 颜色 preset。
 */
export const presetAntdColors = (
  options: PresetAntdColorsOptions = {},
): Preset => {
  const antPrefix = options.antPrefix ?? "ant";
  const themes = options.themes ?? {
    ":root, .light": { algorithm: antdTheme.defaultAlgorithm },
    ".dark": { algorithm: antdTheme.darkAlgorithm },
  };

  const baseToken = antdTheme.getDesignToken(Object.values(themes)[0]);

  type ColorMap = Record<string, string | Record<string, string>>;
  const colors: ColorMap = {};
  const palettes = new Set<string>();

  for (const key of Object.keys(baseToken)) {
    if (/^color[A-Z]/.test(key)) {
      const short = kebab(key.slice("color".length));
      colors[short] = `var(--${antPrefix}-color-${short})`;
      continue;
    }
    const m = PALETTE_RE.exec(key);
    if (m) {
      const [, palette, num] = m;
      palettes.add(palette);
      colors[palette] ??= {};
      (colors[palette] as Record<string, string>)[num] =
        `var(--${antPrefix}-${key})`;
    }
  }
  for (const p of palettes) {
    (colors[p] as Record<string, string>).DEFAULT =
      `var(--${antPrefix}-${p}-6)`;
  }

  const toVarName = (key: string): string => {
    if (/^color[A-Z]/.test(key)) {
      return `--${antPrefix}-color-${kebab(key.slice("color".length))}`;
    }
    return `--${antPrefix}-${PALETTE_RE.test(key) ? key : kebab(key)}`;
  };

  const preflightCss = Object.entries(themes)
    .map(([selector, cfg]) => {
      const lines = Object.entries(antdTheme.getDesignToken(cfg))
        .filter(([, v]) => typeof v === "string" || typeof v === "number")
        .map(([k, v]) => `  ${toVarName(k)}: ${v};`);
      return `${selector} {\n${lines.join("\n")}\n}`;
    })
    .join("\n");

  // wind4 内部生成颜色变量时强制用 `var(--colors-{key})`，
  // 这里在 postprocess 阶段把我们注册过的键改写成 `var(--ant-{key})`，
  // 不动 wind4 自带调色板（`--colors-blue-500` 等）。
  const antVarByFlat = new Map<string, string>();
  for (const k of Object.keys(baseToken)) {
    if (/^color[A-Z]/.test(k)) {
      const short = kebab(k.slice("color".length));
      antVarByFlat.set(short, `var(--${antPrefix}-color-${short})`);
    } else if (PALETTE_RE.test(k)) {
      antVarByFlat.set(k, `var(--${antPrefix}-${k})`);
    }
  }
  for (const p of palettes) {
    antVarByFlat.set(p, `var(--${antPrefix}-${p}-6)`);
  }
  const COLORS_VAR_RE = /var\(--colors-([\w-]+)\)/g;
  const rewriteColorVar = (raw: string): string =>
    raw.replace(
      COLORS_VAR_RE,
      (m, flat: string) => antVarByFlat.get(flat) ?? m,
    );

  return {
    extendTheme: (theme: { colors?: ColorMap }) => {
      theme.colors ??= {};
      const dst = theme.colors;
      for (const [k, v] of Object.entries(colors)) {
        if (typeof v === "string") {
          dst[k] = v;
        } else {
          dst[k] = { ...((dst[k] as Record<string, string>) ?? {}), ...v };
        }
      }
    },
    name: "preset-antd-colors",
    postprocess: (util) => {
      for (const entry of util.entries) {
        if (
          typeof entry[1] === "string" &&
          entry[1].includes("var(--colors-")
        ) {
          entry[1] = rewriteColorVar(entry[1]);
        }
      }
    },
    preflights: [{ getCSS: () => preflightCss }],
  };
};

export default presetAntdColors;
