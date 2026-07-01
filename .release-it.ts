const changelogFile = "CHANGELOG.md";
const versionVariable = "$" + "{version}";

import type { Config } from "release-it";

const config = {
  git: {
    commitMessage: `chore: release v${versionVariable}`,
    getLatestTagFromAllRefs: true,
    tagAnnotation: `Release v${versionVariable}`,
    tagName: `v${versionVariable}`,
  },
  github: {
    release: false,
  },
  npm: {
    publish: false,
  },
  plugins: {
    "@release-it/bumper": {
      out: [
        {
          file: "src-tauri/Cargo.toml",
          path: "package.version",
        },
        {
          file: "src-tauri/Cargo.lock",
          path: "package.0.version",
        },
      ],
    },
    "@release-it/conventional-changelog": {
      ignoreRecommendedBump: true,
      infile: changelogFile,
      preset: {
        name: "conventionalcommits",
        types: [
          { section: "✨ Features", type: "feat" },
          { section: "🐛 Bug Fixes", type: "fix" },
          { section: "⚡️ Performance", type: "perf" },
          { section: "⏪️ Reverts", type: "revert" },
        ],
      },
    },
  },
} satisfies Config;

export default config;
