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
	transformers: [transformerVariantGroup(), transformerDirectives()],
});
