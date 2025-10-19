import {
  defineConfig,
  presetIcons,
  presetUno,
  transformerDirectives,
  transformerVariantGroup,
} from "unocss";

export default defineConfig({
  presets: [presetUno(), presetIcons()],
  rules: [["outline-none", { outline: "none" }]],
  shortcuts: [
    [/^bg-color-(\d+)$/, ([, d]) => `bg-bg-${d}`],
    [/^text-color-(\d+)$/, ([, d]) => `text-text-${d}`],
    [/^b-color-(\d+)$/, ([, d]) => `b-border-${d}`],
    [/^(.*)-primary-(\d+)$/, ([, s, d]) => `${s}-[var(--ant-blue-${d})]`],
  ],
  theme: {
    colors: {
      alipay: "#0c79fe",
      "bg-1": "var(--ant-color-bg-container)",
      "bg-2": "var(--ant-color-bg-layout)",
      "bg-3": "var(--ant-color-fill-quaternary)",
      "bg-4": "var(--ant-color-fill-content)",
      "border-1": "var(--ant-color-border)",
      "border-2": "var(--ant-color-border-secondary)",
      danger: "var(--ant-red)",
      gold: "var(--ant-gold)",
      primary: "var(--ant-blue)",
      qq: "#0099ff",
      success: "var(--ant-green)",
      "text-1": "var(--ant-color-text)",
      "text-2": "var(--ant-color-text-secondary)",
      "text-3": "var(--ant-color-text-tertiary)",
      wechat: "#00c25f",
    },
  },
  transformers: [
    transformerVariantGroup(),
    transformerDirectives({
      applyVariable: ["--uno"],
    }),
  ],
});
