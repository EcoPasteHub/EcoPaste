import { execSync } from "node:child_process";

(() => {
	const platform = process.platform;

	const logoName = platform !== "darwin" ? "logo" : "logo-mac";

	const command = `tauri icon src-tauri/assets/${logoName}.png`;

	execSync(command, { stdio: "inherit" });
})();
