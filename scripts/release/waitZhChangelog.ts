import { existsSync } from "node:fs";
import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { setTimeout as sleep } from "node:timers/promises";

const [version, zhChangelogFile = "CHANGELOG.zh-CN.md"] = process.argv.slice(2);
const placeholder = "<!-- 请补充中文更新日志，保存后发布流程会自动继续。 -->";
const pollMs = Number(process.env.ZH_CHANGELOG_POLL_MS ?? 1000);
const waitMs = Number(process.env.ZH_CHANGELOG_WAIT_MS ?? 30 * 60 * 1000);

if (!version) {
  throw new Error(
    "Usage: tsx scripts/release/waitZhChangelog.ts <version> [zhFile]",
  );
}

const zhChangelogPath = resolve(process.cwd(), zhChangelogFile);

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function findReleaseSection(
  content: string,
  targetVersion: string,
): string | null {
  const escapedVersion = escapeRegExp(targetVersion);
  const headingPattern = new RegExp(
    `^##\\s+\\[?${escapedVersion}\\]?\\b.*$`,
    "m",
  );
  const headingMatch = content.match(headingPattern);

  if (!headingMatch || typeof headingMatch.index !== "number") {
    return null;
  }

  const bodyStart = headingMatch.index + headingMatch[0].length;
  const nextHeadingPattern = /^##\s+/gm;
  nextHeadingPattern.lastIndex = bodyStart;
  const nextHeadingMatch = nextHeadingPattern.exec(content);
  const bodyEnd = nextHeadingMatch?.index ?? content.length;

  return content.slice(bodyStart, bodyEnd).trim();
}

function isReleaseSectionReady(content: string): boolean {
  const section = findReleaseSection(content, version);

  if (!section || section.includes(placeholder)) {
    return false;
  }

  return section.replace(/<!--[\s\S]*?-->/g, "").trim().length > 0;
}

async function readText(path: string): Promise<string> {
  if (!existsSync(path)) {
    return "";
  }

  return await readFile(path, "utf8");
}

function buildInitialContent(existingContent: string): string {
  const releaseBlock = `## [${version}]\n\n${placeholder}`;

  if (!existingContent.trim()) {
    return `# 更新日志\n\n${releaseBlock}\n`;
  }

  const h1Match = existingContent.match(/^# .*(?:\r?\n|$)/);

  if (h1Match?.index !== 0) {
    return `${releaseBlock}\n\n${existingContent.trimStart()}`;
  }

  const bodyStart = h1Match[0].length;
  const title = existingContent.slice(0, bodyStart).trimEnd();
  const rest = existingContent.slice(bodyStart).trimStart();

  return rest
    ? `${title}\n\n${releaseBlock}\n\n${rest}`
    : `${title}\n\n${releaseBlock}\n`;
}

async function ensureReleaseSection(): Promise<void> {
  const content = await readText(zhChangelogPath);

  if (findReleaseSection(content, version) === null) {
    await writeFile(zhChangelogPath, buildInitialContent(content), "utf8");
  }
}

async function waitForReleaseSection(): Promise<void> {
  const start = Date.now();

  while (true) {
    const content = await readText(zhChangelogPath);

    if (isReleaseSectionReady(content)) {
      return;
    }

    if (process.env.CI) {
      throw new Error(
        `${zhChangelogFile} must contain Chinese notes for ${version}.`,
      );
    }

    if (waitMs > 0 && Date.now() - start > waitMs) {
      throw new Error(`Timed out waiting for ${zhChangelogFile} ${version}.`);
    }

    await sleep(pollMs);
  }
}

await ensureReleaseSection();
await waitForReleaseSection();
