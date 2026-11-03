/**
 * 纯文本格式化工具模块
 * 负责自动清洗从电脑或其他设备同步过来的富文本、HTML、RTF 标签、不可见字符等，
 * 保证写入手机系统剪贴板的内容为 100% 干净规范的纯文本。
 */

function decodeHtmlEntities(str: string): string {
  return str
    .replace(/&nbsp;/gi, ' ')
    .replace(/&amp;/gi, '&')
    .replace(/&lt;/gi, '<')
    .replace(/&gt;/gi, '>')
    .replace(/&quot;/gi, '"')
    .replace(/&#39;|&apos;/gi, "'")
    .replace(/&#(\d+);/g, (_match: string, dec: string): string => {
      const code = parseInt(dec, 10);
      return !isNaN(code) && code > 0 ? String.fromCharCode(code) : '';
    })
    .replace(/&#x([0-9a-fA-F]+);/g, (_match: string, hex: string): string => {
      const code = parseInt(hex, 16);
      return !isNaN(code) && code > 0 ? String.fromCharCode(code) : '';
    });
}

function extractHtmlText(html: string): string {
  let s = html;

  // 1. 移除 script, style 与 HTML 注释
  s = s.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '');
  s = s.replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, '');
  s = s.replace(/<!--[\s\S]*?-->/g, '');

  // 2. 块级标签和换行标签转为换行符
  s = s.replace(/<br\s*\/?>/gi, '\n');
  s = s.replace(/<\/(?:p|div|h[1-6]|tr|li|blockquote|section|article)>/gi, '\n');
  s = s.replace(/<(?:p|div|h[1-6]|tr|li|blockquote|section|article)\b[^>]*>/gi, '');

  // 3. 剥离其余所有 HTML 标签
  s = s.replace(/<[^>]+>/g, '');

  // 4. 还原常见 HTML 实体
  return decodeHtmlEntities(s);
}

function extractRtfText(rtf: string): string {
  let s = rtf;

  // 移除 {\fonttbl...}, {\colortbl...}, {\stylesheet...}, {\info...} 等 RTF 头部信息
  s = s.replace(/\{\\(?:fonttbl|colortbl|stylesheet|info)[^}]*\}/gi, '');

  // 替换换行与制表符控制字
  s = s.replace(/\\(?:par|line|cell)\b/gi, '\n');
  s = s.replace(/\\tab\b/gi, '    ');

  // 移除控制字，如 \b, \i, \fs20, \cf1 等
  s = s.replace(/\\[a-zA-Z]+(-?\d+)? ?/g, '');

  // 移除所有成对花括号
  s = s.replace(/[{}]/g, '');

  return s;
}

/**
 * 将任意原始输入文本清洗并标准化为纯文本
 */
export function cleanToPlainText(raw: string): string {
  if (!raw) {
    return '';
  }

  let text = raw;

  // 1. 去除不可见 Unicode 字符（BOM、零宽空格、双向控制符等）
  text = text.replace(/[\u200B-\u200F\uFEFF]/g, '');

  // 2. 替换不换行空格（NBSP）为标准空格
  text = text.replace(/\u00A0/g, ' ');

  // 3. 去除 ANSI 终端转义序列
  text = text.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '');

  // 4. 若为 RTF 格式，剥离 RTF 语法
  if (text.trim().startsWith('{\\rtf')) {
    text = extractRtfText(text);
  }

  // 5. 若包含 HTML 标签或实体，剥离 HTML
  const hasHtml = /<(?:html|body|div|p|span|table|tr|td|ul|ol|li|h[1-6]|b|strong|i|em|br|a|font)\b[^>]*>/i.test(text)
    || /&(?:nbsp|amp|lt|gt|quot|apos|#\d+|#x[0-9a-fA-F]+);/i.test(text);
  if (hasHtml) {
    text = extractHtmlText(text);
  }

  // 6. 统一换行符 \r\n / \r -> \n
  text = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');

  // 7. 去除每行行末的无意义多余空格
  text = text.replace(/[ \t]+$/gm, '');

  // 8. 压缩连续过多的空行（3个及以上压缩为2个）
  text = text.replace(/\n{3,}/g, '\n\n');

  return text.trim();
}
