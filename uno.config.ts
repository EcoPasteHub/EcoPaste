import { defineConfig, presetWind4 } from "unocss";
import { presetAntdColors } from "./src/unocss/presetAntdColors";

export default defineConfig({
  presets: [presetWind4(), presetAntdColors()],
});
