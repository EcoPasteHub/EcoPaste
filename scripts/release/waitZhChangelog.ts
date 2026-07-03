import { createReadStream, createWriteStream, existsSync } from "node:fs";
import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { createInterface } from "node:readline/promises";
import type { Readable, Writable } from "node:stream";

const [version, zhChangelogFile = "CHANGELOG.zh-CN.md"] = process.argv.slice(2);

if (!version) {
  throw new Error(
    "Usage: tsx scripts/release/waitZhChangelog.ts <version> [zhFile]",
  );
}

const zhChangelogPath = resolve(process.cwd(), zhChangelogFile);

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function hasReleaseHeading(content: string, targetVersion: string): boolean {
  const escapedVersion = escapeRegExp(targetVersion);
  return new RegExp(`^##\\s+\\[?${escapedVersion}\\]?\\b.*$`, "m").test(
    content,
  );
}

async function readText(path: string): Promise<string> {
  if (!existsSync(path)) {
    return "";
  }

  return await readFile(path, "utf8");
}

async function ensureFile(): Promise<void> {
  if (!existsSync(zhChangelogPath)) {
    await writeFile(zhChangelogPath, "# 更新日志\n", "utf8");
  }
}

function createPromptIo(): {
  close: () => void;
  input: Readable;
  output: Writable;
} {
  if (process.stdin.isTTY && process.stdout.isTTY) {
    return {
      close: () => {},
      input: process.stdin,
      output: process.stdout,
    };
  }

  const input = createReadStream(
    process.platform === "win32" ? "CONIN$" : "/dev/tty",
  );
  const output = createWriteStream(
    process.platform === "win32" ? "CONOUT$" : "/dev/tty",
  );

  return {
    close: () => {
      input.close();
      output.close();
    },
    input,
    output,
  };
}

async function waitForEnter(): Promise<void> {
  if (process.env.CI) {
    throw new Error(
      `${zhChangelogFile} must be updated before running release-it in CI.`,
    );
  }

  const { close, input, output } = createPromptIo();
  const prompt = createInterface({
    input,
    output,
  });

  try {
    await prompt.question(
      `\nUpdate ${zhChangelogFile} with a ## [${version}] section, then press Enter to continue release-it...`,
    );
  } finally {
    prompt.close();
    close();
  }
}

await ensureFile();
await waitForEnter();

const zhChangelog = await readText(zhChangelogPath);

if (!hasReleaseHeading(zhChangelog, version)) {
  throw new Error(`Missing ## [${version}] section in ${zhChangelogFile}.`);
}
