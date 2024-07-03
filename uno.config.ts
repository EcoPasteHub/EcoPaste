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
	rules: [
		["outline-none", { outline: "none" }],
		[
			/^line-clamp-(\d+)$/,
			(match) => ({
				display: "-webkit-box",
				overflow: "hidden",
				"-webkit-line-clamp": `${match[1]}`,
				"-webkit-box-orient": "vertical",
			}),
		],
	],
	shortcuts: [
		[/^bg-(\d+)$/, ([, d]) => `bg-[var(--color-bg-${d})]`],
		[/^color-(\d+)$/, ([, d]) => `text-[var(--color-text-${d})]`],
		[/^b-color-(\d+)$/, ([, d]) => `b-[var(--color-border-${d})]`],
		[/^(.*)-primary-(\d+)$/, ([, s, d]) => `${s}-[var(--ant-blue-${d})]`],
		{
			"antd-focus":
				"b b-color-1 hover:b-primary-5 transition focus:(shadow-[0_0_0_2px_rgba(5,145,255,0.1)] outline-none b-primary) dark:focus:shadow-[0_0_0_2px_rgba(0,60,180,0.15)]",
		},
	],
	theme: {
		colors: {
			primary: "var(--ant-blue-6)",
			red: "var(--ant-red)",
			gold: "var(--ant-gold)",
		},
	},
});
