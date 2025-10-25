import react from "@vitejs/plugin-react";
import UnoCSS from "unocss/vite";
import { defineConfig } from "vite";

const host = process.env.TAURI_DEV_HOST;

// https://vitejs.dev/config/
export default defineConfig(async () => ({
  build: {
    chunkSizeWarningLimit: 3000,
  },
  // Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
  //
  // 1. prevent vite from obscuring rust errors
  clearScreen: false,
  css: {
    preprocessorOptions: {
      scss: {
        // https://sass-lang.com/documentation/breaking-changes/legacy-js-api/#silencing-warnings
        silenceDeprecations: ["legacy-js-api"],
      },
    },
  },
  plugins: [react(), UnoCSS()],
  resolve: {
    alias: {
      "@": "/src",
    },
  },
  // 2. tauri expects a fixed port, fail if that port is not available
  server: {
    hmr: host
      ? {
          host,
          port: 1421,
          protocol: "ws",
        }
      : undefined,
    host: host || false,
    port: 1420,
    strictPort: true,
    watch: {
      // 3. tell vite to ignore watching `src-tauri`
      ignored: ["**/src-tauri/**"],
    },
  },
}));
