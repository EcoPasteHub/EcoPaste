import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import emojiMartData from '@emoji-mart/data' assert { type: 'json' };

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function main() {
  const rootDir = path.resolve(__dirname, '..');
  const zhEmojiFulPath = path.resolve(rootDir, 'zh-emoji-ful.json');
  const cldrPath = path.resolve(rootDir, 'cldr.json');
  const outPath = path.resolve(rootDir, 'src', 'assets', 'emoji-data.json');

  const zhEmojiFul = JSON.parse(await fs.readFile(zhEmojiFulPath, 'utf-8'));
  const cldr = JSON.parse(await fs.readFile(cldrPath, 'utf-8'));
  
  // 合并 cldr.json 和 zh-emoji-ful.json
  const annotations = {
    ...(cldr.annotations?.annotations || {}),
    ...(zhEmojiFul.annotations?.annotations || {}),
    ...(zhEmojiFul.annotationsDerived?.annotations || {}),
  };

  const newEmojis: Record<string, any> = {};

  for (const [id, emoji] of Object.entries(emojiMartData.emojis)) {
    const native = emoji.skins[0]?.native;
    
    // 尝试从中文包获取数据
    let zhInfo = native ? annotations[native] : null;
    if (!zhInfo && native) {
      const fallbackNative = native.replace(/\uFE0F/g, '');
      zhInfo = annotations[fallbackNative];
    }
    
    // 提取中文名 (tts 通常包含完整名称，比如 "挥手: 较浅肤色"，我们可以截取冒号前面的部分作为基础名)
    let zhName = emoji.name;
    if (zhInfo?.tts && zhInfo.tts[0]) {
      zhName = zhInfo.tts[0].split(':')[0].trim();
    }
    
    // 提取中文关键词
    let zhKeywords = emoji.keywords;
    if (zhInfo?.default) {
      zhKeywords = Array.from(new Set([...emoji.keywords, ...zhInfo.default]));
    }

    newEmojis[id] = {
      ...emoji,
      name: zhName,
      keywords: zhKeywords,
    };
  }

  const newData = {
    ...emojiMartData,
    emojis: newEmojis,
  };

  // Ensure assets dir exists
  await fs.mkdir(path.dirname(outPath), { recursive: true });
  
  await fs.writeFile(outPath, JSON.stringify(newData, null, 2), 'utf-8');
  console.log(`Successfully built emoji data to ${outPath}`);
}

main().catch(console.error);
