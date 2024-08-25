import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { name, version } from "../package.json";

const __dirname = dirname(fileURLToPath(import.meta.url));

const updateReadmeVersion = () => {
	const dirPath = resolve(__dirname, "..");

	const files = readdirSync(dirPath).filter((file) => {
		return file.startsWith("README.");
	});

	for (const file of files) {
		const path = resolve(dirPath, file);

		let content = readFileSync(path, "utf-8");

		content = content.replace(/(\d+\.\d+\.\d+)/g, version);

		writeFileSync(path, content);
	}
};

const updateCargoVersion = () => {
	const tomlPath = resolve(__dirname, "..", "src-tauri", "Cargo.toml");
	const lockPath = resolve(__dirname, "..", "src-tauri", "Cargo.lock");

	for (const path of [tomlPath, lockPath]) {
		let content = readFileSync(path, "utf-8");

		const regexp = new RegExp(
			`(name\\s*=\\s*"${name}"\\s*version\\s*=\\s*)"\\d+.\\d+.\\d+"`,
		);

		content = content.replace(regexp, `$1"${version}"`);

		writeFileSync(path, content);
	}
};

updateReadmeVersion();
updateCargoVersion();
