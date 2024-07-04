import type { Config } from "release-it";

export default {
	git: {
		commitMessage: "chore: release v${version}",
		tagName: "v${version}",
	},
	npm: {
		publish: false,
	},
	hooks: {
		"after:bump": "ts-node scripts/release.ts",
	},
} satisfies Config;
