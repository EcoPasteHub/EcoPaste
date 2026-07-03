import { closeSync, existsSync, openSync, readSync, writeSync } from "node:fs";
import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

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

function sleepSync(ms: number): void {
  Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);
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

function openTerminalFd(kind: "input" | "output"): {
  close: () => void;
  fd: number;
} {
  if (kind === "input" && process.stdin.isTTY) {
    return {
      close: () => {},
      fd: process.stdin.fd,
    };
  }

  if (kind === "output" && process.stdout.isTTY) {
    return {
      close: () => {},
      fd: process.stdout.fd,
    };
  }

  const path =
    process.platform === "win32"
      ? kind === "input"
        ? "CONIN$"
        : "CONOUT$"
      : "/dev/tty";
  const fd = openSync(path, kind === "input" ? "r" : "w");

  return {
    close: () => closeSync(fd),
    fd,
  };
}

function waitForEnter(): void {
  if (process.env.CI) {
    throw new Error(
      `${zhChangelogFile} must be updated before running release-it in CI.`,
    );
  }

  const input = openTerminalFd("input");
  const output = openTerminalFd("output");
  const buffer = Buffer.alloc(1);

  try {
    writeSync(
      output.fd,
      `\nUpdate ${zhChangelogFile} with a ## [${version}] section, then press Enter to continue release-it...`,
    );

    while (true) {
      let bytesRead = 0;

      try {
        bytesRead = readSync(input.fd, buffer, 0, 1, null);
      } catch (error) {
        if (
          error instanceof Error &&
          "code" in error &&
          error.code === "EAGAIN"
        ) {
          sleepSync(50);
          continue;
        }

        throw error;
      }

      if (bytesRead === 0 || buffer[0] === 10 || buffer[0] === 13) {
        break;
      }

      if (buffer[0] === 3) {
        throw new Error(
          "Release cancelled while waiting for Chinese changelog.",
        );
      }
    }

    writeSync(output.fd, "\n");
  } finally {
    input.close();
    output.close();
  }
}

await ensureFile();
waitForEnter();

const zhChangelog = await readText(zhChangelogPath);

if (!hasReleaseHeading(zhChangelog, version)) {
  throw new Error(`Missing ## [${version}] section in ${zhChangelogFile}.`);
}
