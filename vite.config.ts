import react from "@vitejs/plugin-react";
import UnoCSS from "unocss/vite";
import AutoImport from "unplugin-auto-import/vite";
import { defineConfig } from "vite";

// https://vitejs.dev/config/
export default defineConfig(async () => ({
	plugins: [
		react(),
		UnoCSS(),
		AutoImport({
			imports: ["react", "ahooks", "react-router-dom", "react-i18next"],
			dts: "src/types/auto-imports.d.ts",
			dirs: [
				"src/router",
				"src/utils",
				"src/stores",
				"src/database",
				"src/hooks",
				"src/constants",
				"src/plugins",
				"src/locales",
			],
		}),
	],
	resolve: {
		alias: {
			"@": "/src",
		},
	},
	// Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
	//
	// 1. prevent vite from obscuring rust errors
	clearScreen: false,
	// 2. tauri expects a fixed port, fail if that port is not available
	server: {
		port: 1420,
		strictPort: true,
		watch: {
			// 3. tell vite to ignore watching `src-tauri`
			ignored: ["**/src-tauri/**"],
		},
	},
	build: {
		chunkSizeWarningLimit: 3000,
	},
}));
