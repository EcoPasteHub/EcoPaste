import presetRemToPx from "@unocss/preset-rem-to-px";
import {
	defineConfig,
	presetIcons,
	presetUno,
	transformerDirectives,
	transformerVariantGroup,
} from "unocss";

export default defineConfig({
	presets: [
		presetUno(),
		presetIcons(),
		presetRemToPx({
			baseFontSize: 4,
		}),
	],
	transformers: [
		transformerVariantGroup(),
		transformerDirectives({
			applyVariable: ["--uno"],
		}),
	],
	rules: [["outline-none", { outline: "none" }]],
	shortcuts: [
		[/^bg-(\d+)$/, ([, d]) => `bg-[var(--color-bg-${d})]`],
		[/^color-(\d+)$/, ([, d]) => `text-[var(--color-text-${d})]`],
		[/^b-color-(\d+)$/, ([, d]) => `b-[var(--color-border-${d})]`],
		[/^(.*)-primary-(\d+)$/, ([, s, d]) => `${s}-[var(--ant-blue-${d})]`],
	],
	theme: {
		colors: {
			primary: "var(--ant-blue)",
			success: "var(--ant-green)",
			danger: "var(--ant-red)",
			gold: "var(--ant-gold)",
		},
	},
});
