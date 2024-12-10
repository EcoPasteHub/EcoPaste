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
		[/^bg-color-(\d+)$/, ([, d]) => `bg-bg-${d}`],
		[/^text-color-(\d+)$/, ([, d]) => `text-text-${d}`],
		[/^b-color-(\d+)$/, ([, d]) => `b-border-${d}`],
		[/^(.*)-primary-(\d+)$/, ([, s, d]) => `${s}-[var(--ant-blue-${d})]`],
		{
			"antd-input":
				"b hover:b-primary-5 transition outline-none focus:antd-input-focus",
			"antd-input-focus":
				"b-primary shadow-[0_0_0_2px_rgba(5,145,255,0.1)] dark:shadow-[0_0_0_2px_rgba(0,60,180,0.15)]",
		},
	],
	theme: {
		colors: {
			"bg-1": "var(--ant-color-bg-container)",
			"bg-2": "var(--ant-color-bg-layout)",
			"bg-3": "var(--ant-color-fill-quaternary)",
			"bg-4": "var(--ant-color-fill-content)",
			"text-1": "var(--ant-color-text)",
			"text-2": "var(--ant-color-text-secondary)",
			"text-3": "var(--ant-color-text-tertiary)",
			"border-1": "var(--ant-color-border)",
			"border-2": "var(--ant-color-border-secondary)",
			primary: "var(--ant-blue)",
			success: "var(--ant-green)",
			danger: "var(--ant-red)",
			gold: "var(--ant-gold)",
			qq: "#0099ff",
			wechat: "#00c25f",
			alipay: "#0c79fe",
		},
	},
});
