import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

const readPackageJson = () => {
	const path = resolve(__dirname, "..", "package.json");

	const content = readFileSync(path, "utf-8");

	const { name, version } = JSON.parse(content);

	return {
		name,
		version,
	};
};

const updateReadmeVersion = () => {
	const { version } = readPackageJson();

	const path = resolve(__dirname, "..", "README.md");

	let content = readFileSync(path, "utf-8");

	content = content.replace(/(\d+\.\d+\.\d+)/g, version);

	writeFileSync(path, content);
};

const updateCargoVersion = () => {
	const { name, version } = readPackageJson();

	const tomlPath = resolve(__dirname, "..", "src-tauri", "Cargo.toml");
	const lockPath = resolve(__dirname, "..", "src-tauri", "Cargo.lock");

	for (const path of [tomlPath, lockPath]) {
		let content = readFileSync(path, "utf-8");

		const reg = new RegExp(
			`(name\\s*=\\s*"${name}"\\s*version\\s*=\\s*)"\\d+.\\d+.\\d+"`,
		);

		content = content.replace(reg, `$1"${version}"`);

		writeFileSync(path, content);
	}
};

updateReadmeVersion();
updateCargoVersion();
