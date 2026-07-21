const changelogFile = "CHANGELOG.md";
const zhChangelogFile = "CHANGELOG.zh-CN.md";
const versionVariable = "$" + "{version}";
const nightlyVersionMarker = "-nightly.";

import type { Config } from "release-it";

type ReleaseItHooks = NonNullable<Config["hooks"]> & Record<string, string>;

const hooks: ReleaseItHooks = {
  "after:@release-it/conventional-changelog:beforeRelease": `bash -c 'printf "\\nCHANGELOG.md has been generated. Update ${zhChangelogFile}, then press Enter to continue release-it..." > /dev/tty; IFS= read -r -s -n 1 _ < /dev/tty; printf "\\n" > /dev/tty'`,
};

const config = {
  git: {
    commitMessage: `chore: release v${versionVariable}`,
    getLatestTagFromAllRefs: false,
    tagAnnotation: `Release v${versionVariable}`,
    tagExclude: "v*-nightly.*",
    tagName: `v${versionVariable}`,
  },
  github: {
    release: false,
  },
  hooks,
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
          { section: "⏪ Reverts", type: "revert" },
        ],
      },
      writerOpts: {
        generateOn: (commit: { version?: string }) => {
          return Boolean(
            commit.version && !commit.version.includes(nightlyVersionMarker),
          );
        },
      },
    },
  },
} satisfies Config;

export default config;
