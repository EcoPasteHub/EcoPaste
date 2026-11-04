#!/usr/bin/env node
/**
 * EcoPaste 鸿蒙端版本号自动维护脚本
 * 自动递增 harmony/AppScope/app.json5 中的 versionCode 和 versionName，
 * 以及 harmony/entry/oh-package.json5 中的 version。
 */

const fs = require('node:fs');
const path = require('node:path');

const HARMONY_DIR = __dirname;
const APP_JSON5 = path.join(HARMONY_DIR, 'AppScope', 'app.json5');
const ENTRY_OH_PKG = path.join(HARMONY_DIR, 'entry', 'oh-package.json5');

function readFiles() {
  if (!fs.existsSync(APP_JSON5)) {
    throw new Error(`找不到文件: ${APP_JSON5}`);
  }
  const appContent = fs.readFileSync(APP_JSON5, 'utf8');
  const entryContent = fs.existsSync(ENTRY_OH_PKG) ? fs.readFileSync(ENTRY_OH_PKG, 'utf8') : null;
  return { appContent, entryContent };
}

function parseCurrentVersion(appContent) {
  const codeMatch = appContent.match(/"versionCode"\s*:\s*(\d+)/);
  const nameMatch = appContent.match(/"versionName"\s*:\s*"([^"]+)"/);

  if (!codeMatch || !nameMatch) {
    throw new Error('无法在 app.json5 中解析 versionCode 或 versionName');
  }

  return {
    versionCode: parseInt(codeMatch[1], 10),
    versionName: nameMatch[1],
  };
}

function bumpVersionName(currentName, type = 'patch') {
  const parts = currentName.split('.').map((p) => parseInt(p, 10));
  while (parts.length < 3) parts.push(0);

  if (type === 'major') {
    parts[0] += 1;
    parts[1] = 0;
    parts[2] = 0;
  } else if (type === 'minor') {
    parts[1] += 1;
    parts[2] = 0;
  } else {
    // patch
    parts[2] += 1;
  }

  return parts.join('.');
}

function main() {
  const args = process.argv.slice(2);
  const { appContent, entryContent } = readFiles();
  const current = parseCurrentVersion(appContent);

  if (args.includes('--get')) {
    process.stdout.write(current.versionName);
    return;
  }
  if (args.includes('--get-code')) {
    process.stdout.write(String(current.versionCode));
    return;
  }

  let bumpType = 'patch';
  let targetVersion = null;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--major') bumpType = 'major';
    else if (args[i] === '--minor') bumpType = 'minor';
    else if (args[i] === '--patch') bumpType = 'patch';
    else if (args[i] === '--set' && args[i + 1]) {
      targetVersion = args[i + 1];
      i++;
    }
  }

  const newVersionName = targetVersion || bumpVersionName(current.versionName, bumpType);
  const newVersionCode = current.versionCode + 1;

  // 替换 app.json5
  const updatedAppContent = appContent
    .replace(/"versionCode"\s*:\s*\d+/, `"versionCode": ${newVersionCode}`)
    .replace(/"versionName"\s*:\s*"[^"]+"/, `"versionName": "${newVersionName}"`);

  fs.writeFileSync(APP_JSON5, updatedAppContent, 'utf8');

  // 替换 entry/oh-package.json5
  if (entryContent) {
    const updatedEntryContent = entryContent.replace(
      /"version"\s*:\s*"[^"]+"/,
      `"version": "${newVersionName}"`
    );
    fs.writeFileSync(ENTRY_OH_PKG, updatedEntryContent, 'utf8');
  }

  console.log(`[Version] 鸿蒙端版本号已自动递增:`);
  console.log(`  versionName: ${current.versionName} -> ${newVersionName}`);
  console.log(`  versionCode: ${current.versionCode} -> ${newVersionCode}`);
}

main();
