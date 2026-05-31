import { defineConfig, presetWind4 } from "unocss";
import { presetAntdColors } from "./src/unocss/presetAntdColors";

/**
 * 颜色类名（`color-primary` / `bg-bg-container` / `b-border-secondary` 等）由
 * presetAntdColors 直接映射到 antd v6 CSS 变量（`--ant-color-*`），主题切换由
 * antd ConfigProvider 在运行时切换变量值，UnoCSS 不感知具体色值。
 *
 * presetWind4 仍负责 spacing / typography / flex / layout 等非颜色原子类。
 * theme.colors 仅保留 presetWind4 的颜色派生工具（`divide-*` / `ring-*` 等）
 * 需要的最小条目，写法上仍优先用 preset 提供的 `c-/bg-/b-` 短名。
 */
export default defineConfig({
  presets: [presetWind4(), presetAntdColors()],
  theme: {
    colors: {
      "border-secondary": "var(--ant-color-border-secondary)",
    },
  },
});
