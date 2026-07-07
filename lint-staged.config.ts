import type { Configuration } from "lint-staged";

const config: Configuration = {
  "**/*.{ts,tsx,js,jsx,json,jsonc,css}": [
    "biome check --write --no-errors-on-unmatched",
  ],
  "src-tauri/**/*.{rs,toml,lock}": () => {
    return [
      "cargo fmt --manifest-path src-tauri/Cargo.toml --all",
      "cargo clippy --manifest-path src-tauri/Cargo.toml --all-targets --all-features -- -D warnings",
    ];
  },
};

export default config;
