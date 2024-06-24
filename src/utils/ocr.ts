import { Command } from "@tauri-apps/api/shell";

export function parse(path: string): Promise<string> {
	return new Promise((resolve) => {
		let result = "";
		const args = [path, "zh"];
		const command = Command.sidecar("bin/ocr", args);

		command.stdout.on("data", (line) => {
			result = line.toString().trim();
		});

		command.on("close", () => {
			resolve(result);
		});

		command.on("error", (error) => {
			console.error("识别出错", error);
		});

		command.spawn();
	});
}
