import type { EmojiMartData } from "@emoji-mart/data";
import { Tag } from "antd";
import { chunk } from "es-toolkit";
import { useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";
import bootstrapKaomojiData from "../../../../../../../bootstrap-data-merged.json";
import emojiData from "@/assets/emoji-data.json";
import Scrollbar from "@/components/Scrollbar";
import { MainContext } from "@/pages/Main";
import { copyTextToClipboard, pasteTextToClipboard } from "@/plugins/clipboard";
import { scrollElementToCenter } from "@/utils/dom";

const typedData = emojiData as EmojiMartData;
const getEmojiImageSrc = (unified?: string) => {
  return unified ? `/emojipng/${unified.toLowerCase()}.png` : void 0;
};

const EMOJI_CATEGORIES = [
  { id: "frequent", name: "最近使用" },
  { id: "people", name: "表情与角色" },
  { id: "nature", name: "自然与动物" },
  { id: "foods", name: "食物与饮料" },
  { id: "activity", name: "活动" },
  { id: "places", name: "旅行与地点" },
  { id: "objects", name: "物体" },
  { id: "symbols", name: "符号" },
  { id: "flags", name: "旗帜" },
];

interface KaomojiItem {
  id: string;
  text: string;
  keywords: string[];
}

interface SymbolCategory {
  id: string;
  name: string;
  keywords: string[];
  items: KaomojiItem[];
}

interface BootstrapKaomojiData {
  kaomojis?: Array<{
    text?: string;
    keywords?: string;
  }>;
}

const typedBootstrapKaomojiData = bootstrapKaomojiData as BootstrapKaomojiData;

const BASE_KAOMOJI_CATEGORIES: SymbolCategory[] = [
  {
    id: "common",
    items: [
      { id: "common-1", keywords: ["常用", "微笑", "开心"], text: "(●'◡'●)" },
      { id: "common-2", keywords: ["常用", "加油", "打气"], text: "(๑•̀ㅂ•́)و" },
      { id: "common-3", keywords: ["常用", "害羞", "脸红"], text: "(*/ω＼*)" },
      { id: "common-4", keywords: ["常用", "喜欢", "爱心"], text: "(´▽`ʃ♡ƪ)" },
      {
        id: "common-5",
        keywords: ["常用", "耸肩", "无奈"],
        text: "¯\\_(ツ)_/¯",
      },
      {
        id: "common-6",
        keywords: ["常用", "掀桌", "生气"],
        text: "(╯°□°）╯︵ ┻━┻",
      },
    ],
    keywords: ["常用", "classic", "frequent"],
    name: "常用",
  },
  {
    id: "happy",
    items: [
      { id: "happy-1", keywords: ["开心", "高兴", "挥手"], text: "ヽ(✿ﾟ▽ﾟ)ノ" },
      { id: "happy-2", keywords: ["开心", "兴奋", "雀跃"], text: "ヾ(≧▽≦*)o" },
      { id: "happy-3", keywords: ["开心", "满足", "笑"], text: "(●ˇ∀ˇ●)" },
      { id: "happy-4", keywords: ["开心", "打招呼", "招手"], text: "(≧∇≦)ﾉ" },
      { id: "happy-5", keywords: ["开心", "偷笑", "可爱"], text: "(^///^)" },
      { id: "happy-6", keywords: ["开心", "音乐", "轻松"], text: "♪(^∇^*)" },
    ],
    keywords: ["开心", "快乐", "happy"],
    name: "开心",
  },
  {
    id: "love",
    items: [
      { id: "love-1", keywords: ["喜欢", "爱", "心动"], text: "(❤ ω ❤)" },
      { id: "love-2", keywords: ["喜欢", "爱心", "贴贴"], text: "(´▽`ʃ♡ƪ)" },
      {
        id: "love-3",
        keywords: ["喜欢", "拥抱", "亲亲"],
        text: "(づ￣ 3￣)づ",
      },
      { id: "love-4", keywords: ["喜欢", "亲吻", "爱"], text: "( ˘ 3˘)♥" },
      { id: "love-5", keywords: ["喜欢", "温柔", "满足"], text: "(๑˘︶˘๑)" },
      { id: "love-6", keywords: ["喜欢", "可爱", "爱慕"], text: "♡(ӦｖӦ｡)" },
    ],
    keywords: ["喜欢", "爱", "love"],
    name: "喜欢",
  },
  {
    id: "shy",
    items: [
      { id: "shy-1", keywords: ["害羞", "脸红", "捂脸"], text: "(*/ω＼*)" },
      { id: "shy-2", keywords: ["害羞", "偷看", "不好意思"], text: "(*/▽＼*)" },
      { id: "shy-3", keywords: ["害羞", "回避", "不好意思"], text: "(/▽＼)" },
      {
        id: "shy-4",
        keywords: ["害羞", "脸红", "心动"],
        text: "(⁄ ⁄•⁄ω⁄•⁄ ⁄)",
      },
      { id: "shy-5", keywords: ["害羞", "可爱", "红脸"], text: "(⁄⁄•⁄ω⁄•⁄⁄)" },
      { id: "shy-6", keywords: ["害羞", "低头", "拘谨"], text: "(。_。)" },
    ],
    keywords: ["害羞", "脸红", "shy"],
    name: "害羞",
  },
  {
    id: "helpless",
    items: [
      {
        id: "helpless-1",
        keywords: ["无语", "耸肩", "无奈"],
        text: "¯\\_(ツ)_/¯",
      },
      {
        id: "helpless-2",
        keywords: ["无语", "沉默", "嫌弃"],
        text: "(ー_ー)!!",
      },
      {
        id: "helpless-3",
        keywords: ["无语", "怀疑", "斜眼"],
        text: "(￢_￢;)",
      },
      { id: "helpless-4", keywords: ["无语", "尴尬", "震惊"], text: "(⊙_⊙;)" },
      {
        id: "helpless-5",
        keywords: ["无语", "为难", "汗颜"],
        text: "(￣△￣；)",
      },
      {
        id: "helpless-6",
        keywords: ["无语", "轻蔑", "淡定"],
        text: "(￣_,￣ )",
      },
    ],
    keywords: ["无语", "无奈", "helpless"],
    name: "无语",
  },
  {
    id: "angry",
    items: [
      {
        id: "angry-1",
        keywords: ["生气", "掀桌", "愤怒"],
        text: "(╯°□°）╯︵ ┻━┻",
      },
      { id: "angry-2", keywords: ["生气", "凶", "不满"], text: "(▼へ▼メ)" },
      { id: "angry-3", keywords: ["生气", "暴怒", "抓狂"], text: "٩(╬ʘ益ʘ╬)۶" },
      { id: "angry-4", keywords: ["生气", "发火", "怒"], text: "(#`O′)" },
      { id: "angry-5", keywords: ["生气", "不爽", "瞪眼"], text: "(｀д′)" },
      {
        id: "angry-6",
        keywords: ["生气", "骂人", "愤怒"],
        text: "凸(艹皿艹 )",
      },
    ],
    keywords: ["生气", "愤怒", "angry"],
    name: "生气",
  },
];

const KAOMOJI_CATEGORY_RULES = [
  {
    id: "angry",
    keywords: [
      "angry",
      "disapproval",
      "disgust",
      "frustrated",
      "furious",
      "mad",
      "tableflip",
      "upset",
      "厌恶",
      "心烦",
      "愤怒",
      "掀桌",
      "沮丧",
      "生气",
      "疯狂",
      "翻转",
    ],
  },
  {
    id: "love",
    keywords: [
      "embrace",
      "heart",
      "hug",
      "infatuated",
      "kiss",
      "love",
      "亲亲",
      "亲吻",
      "喜欢",
      "心动",
      "爱",
      "爱心",
      "拥抱",
      "贴贴",
      "迷恋",
    ],
  },
  {
    id: "shy",
    keywords: [
      "embarrassed",
      "hide",
      "hiding",
      "peek",
      "peeking",
      "shy",
      "不好意思",
      "偷看",
      "回避",
      "害羞",
      "捂脸",
      "脸红",
      "脸熱",
    ],
  },
  {
    id: "happy",
    keywords: [
      "excited",
      "greetings",
      "happy",
      "hello",
      "hey",
      "hi",
      "joy",
      "praise",
      "smile",
      "wave",
      "wink",
      "兴奋",
      "开心",
      "快乐",
      "微笑",
      "招手",
      "挥手",
      "打招呼",
      "笑",
      "赞美",
      "问候",
      "高兴",
    ],
  },
  {
    id: "helpless",
    keywords: [
      "apologize",
      "calm",
      "confused",
      "doubt",
      "forlorn",
      "frightened",
      "indifferent",
      "meh",
      "putting",
      "relax",
      "sad",
      "scared",
      "skeptical",
      "sorry",
      "sorrow",
      "surprised",
      "tableunflip",
      "teary-eyed",
      "unsure",
      "whatever",
      "what",
      "抱歉",
      "放回",
      "放松",
      "无奈",
      "无所谓",
      "无语",
      "道歉",
      "冷漠",
      "冷静",
      "含泪",
      "困惑",
      "回来",
      "孤独",
      "害怕",
      "怀疑",
      "悲伤",
      "悲痛",
      "惊恐",
      "惊讶",
      "放回桌子",
      "震惊",
      "难过",
    ],
  },
] as const satisfies Array<{
  id: SymbolCategory["id"];
  keywords: string[];
}>;

const normalizeKaomojiText = (value?: string) => {
  return value?.trim();
};

const normalizeKaomojiKeywords = (value?: string) => {
  return Array.from(
    new Set(
      value
        ?.split(/\s+/)
        .map((entry) => entry.trim())
        .filter(Boolean) ?? [],
    ),
  );
};

const getKaomojiTextLength = (value: string) => {
  return Array.from(value).length;
};

const SHORT_KAOMOJI_MAX_LENGTH = 8;
const MEDIUM_KAOMOJI_MAX_LENGTH = 14;

const getShortKaomojiColumns = (width: number) => {
  return width >= 1080 ? 6 : 5;
};

const getKaomojiRowColumns = (value: string, shortColumns: number) => {
  const length = getKaomojiTextLength(value);

  if (length <= SHORT_KAOMOJI_MAX_LENGTH) {
    return shortColumns;
  }

  if (length <= MEDIUM_KAOMOJI_MAX_LENGTH) {
    return 3;
  }

  return 2;
};

const getKaomojiItemTextClassName = (value: string, columns: number) => {
  const length = getKaomojiTextLength(value);

  if (columns >= 5) {
    return "text-sm";
  }

  if (columns === 3) {
    return length >= 12 ? "text-sm" : "text-base";
  }

  if (length >= 36) {
    return "text-[11px]";
  }

  if (length >= 26) {
    return "text-xs";
  }

  if (length >= 14) {
    return "text-sm";
  }

  return "text-base";
};

const chunkKaomojiItems = (items: KaomojiItem[], shortColumns: number) => {
  const rows: Array<{ items: KaomojiItem[]; columns: number }> = [];
  const groups = new Map<number, KaomojiItem[]>();
  const orderedColumns = [shortColumns, 3, 2];

  for (const item of items) {
    const columns = getKaomojiRowColumns(item.text, shortColumns);
    const group = groups.get(columns);

    if (group) {
      group.push(item);
      continue;
    }

    groups.set(columns, [item]);
  }

  orderedColumns.forEach((columns) => {
    const group = groups.get(columns);
    if (!group?.length) return;

    chunk(group, columns).forEach((itemsInRow) => {
      rows.push({
        columns,
        items: itemsInRow,
      });
    });
  });

  return rows;
};

const getKaomojiCategoryId = (text: string, keywords: string[]) => {
  const haystack = [text, ...keywords].join(" ").toLowerCase();

  for (const rule of KAOMOJI_CATEGORY_RULES) {
    if (rule.keywords.some((keyword) => haystack.includes(keyword.toLowerCase()))) {
      return rule.id;
    }
  }

  return "common";
};

const KAOMOJI_CATEGORIES: SymbolCategory[] = (() => {
  const categories = BASE_KAOMOJI_CATEGORIES.map((category) => {
    return {
      ...category,
      items: [...category.items],
    };
  });
  const categoryMap = new Map(categories.map((category) => [category.id, category]));
  const seenTexts = new Set(
    categories.flatMap((category) => {
      return category.items.map((item) => normalizeKaomojiText(item.text));
    }),
  );

  for (const entry of typedBootstrapKaomojiData.kaomojis ?? []) {
    const text = normalizeKaomojiText(entry.text);
    if (!text || seenTexts.has(text)) continue;

    const keywords = normalizeKaomojiKeywords(entry.keywords);
    const categoryId = getKaomojiCategoryId(text, keywords);
    const category = categoryMap.get(categoryId);
    if (!category) continue;

    category.items.push({
      id: `${category.id}-${category.items.length + 1}`,
      keywords,
      text,
    });
    seenTexts.add(text);
  }

  return categories;
})();

const isVisibleChar = (value: string) => {
  return value.length > 0 && !/[\p{C}\s]/u.test(value);
};

const buildSymbolItems = (
  prefix: string,
  ranges: Array<[number, number]>,
  matcher: (value: string) => boolean,
  keywords: string[],
  extras: string[] = [],
) => {
  const seen = new Set<string>();
  const items: KaomojiItem[] = [];

  const append = (value: string) => {
    if (seen.has(value) || !isVisibleChar(value) || !matcher(value)) return;

    const codePoint = value.codePointAt(0);
    if (!codePoint) return;

    seen.add(value);
    items.push({
      id: `${prefix}-${codePoint.toString(16)}`,
      keywords,
      text: value,
    });
  };

  extras.forEach(append);

  for (const [start, end] of ranges) {
    for (let codePoint = start; codePoint <= end; codePoint += 1) {
      append(String.fromCodePoint(codePoint));
    }
  }

  return items;
};

const SYMBOL_CATEGORIES: SymbolCategory[] = [
  {
    id: "punctuation",
    items: buildSymbolItems(
      "symbol-punctuation",
      [
        [0x0021, 0x002f],
        [0x003a, 0x0040],
        [0x005b, 0x0060],
        [0x007b, 0x007e],
        [0x00a1, 0x00bf],
        [0x2000, 0x206f],
        [0x2e00, 0x2e7f],
        [0x3000, 0x303f],
        [0xff01, 0xff0f],
        [0xff1a, 0xff20],
        [0xff3b, 0xff40],
        [0xff5b, 0xff65],
      ],
      (value) => /\p{P}/u.test(value) || "•·…※†‡‰‱′″‴‵‽".includes(value),
      ["常规标点", "标点", "punctuation"],
      ["…", "—", "–", "•", "·", "※", "′", "″", "‽"],
    ),
    keywords: ["常规标点", "标点", "punctuation"],
    name: "常规标点",
  },
  {
    id: "currency",
    items: buildSymbolItems(
      "symbol-currency",
      [
        [0x0024, 0x0024],
        [0x00a2, 0x00a5],
        [0x058f, 0x058f],
        [0x060b, 0x060b],
        [0x09f2, 0x09f3],
        [0x09fb, 0x09fb],
        [0x0af1, 0x0af1],
        [0x0bf9, 0x0bf9],
        [0x0e3f, 0x0e3f],
        [0x17db, 0x17db],
        [0x20a0, 0x20cf],
        [0xa838, 0xa838],
        [0xfdfc, 0xfdfc],
        [0xfe69, 0xfe69],
        [0xff04, 0xff04],
        [0xffe0, 0xffe6],
      ],
      (value) => /\p{Sc}/u.test(value) || value === "¤",
      ["货币符号", "货币", "currency"],
      ["$", "¢", "£", "¤", "¥", "€", "₹", "₩", "₽", "₿"],
    ),
    keywords: ["货币符号", "货币", "currency"],
    name: "货币符号",
  },
  {
    id: "latin",
    items: buildSymbolItems(
      "symbol-latin",
      [
        [0x00c0, 0x024f],
        [0x1e00, 0x1eff],
      ],
      (value) => /\p{Script=Latin}/u.test(value),
      ["拉丁符号", "拉丁", "latin"],
    ),
    keywords: ["拉丁符号", "拉丁", "latin"],
    name: "拉丁符号",
  },
  {
    id: "geometric",
    items: buildSymbolItems(
      "symbol-geometric",
      [
        [0x25a0, 0x25ff],
        [0x2b12, 0x2bff],
      ],
      (value) => /\p{S}/u.test(value),
      ["几何符号", "几何", "geometric"],
      ["■", "□", "▲", "△", "◆", "◇", "●", "○", "★", "☆"],
    ),
    keywords: ["几何符号", "几何", "geometric"],
    name: "几何符号",
  },
  {
    id: "math",
    items: buildSymbolItems(
      "symbol-math",
      [
        [0x00ac, 0x00ac],
        [0x00b1, 0x00b1],
        [0x00d7, 0x00d7],
        [0x00f7, 0x00f7],
        [0x2070, 0x209f],
        [0x2200, 0x22ff],
        [0x27c0, 0x27ef],
        [0x2980, 0x29ff],
        [0x2a00, 0x2aff],
      ],
      (value) => /\p{Sm}/u.test(value) || "∞≈≠≤≥±×÷√∑∏∫∂∆∇".includes(value),
      ["数学符号", "数学", "math"],
      ["±", "×", "÷", "√", "∞", "≈", "≠", "≤", "≥", "∑", "∏", "∫"],
    ),
    keywords: ["数学符号", "数学", "math"],
    name: "数学符号",
  },
  {
    id: "supplemental",
    items: buildSymbolItems(
      "symbol-supplemental",
      [
        [0x2190, 0x21ff],
        [0x2300, 0x23ff],
        [0x2460, 0x24ff],
        [0x2580, 0x259f],
        [0x2600, 0x26ff],
        [0x2700, 0x27bf],
      ],
      (value) => /\p{S}/u.test(value),
      ["补充符号", "补充", "supplemental"],
      ["←", "→", "↑", "↓", "↔", "⌘", "⌛", "①", "☀", "☑", "✓", "✕"],
    ),
    keywords: ["补充符号", "补充", "supplemental"],
    name: "补充符号",
  },
  {
    id: "language",
    items: buildSymbolItems(
      "symbol-language",
      [
        [0x0370, 0x03ff],
        [0x0400, 0x04ff],
        [0x3000, 0x303f],
      ],
      (value) => {
        return (
          /\p{Script=Greek}/u.test(value) ||
          /\p{Script=Cyrillic}/u.test(value) ||
          /[々〆〤〇〒〓〠〳〴〵、。「」『』【】《》〈〉〔〕〖〗〘〙〚〛〜]/u.test(
            value,
          )
        );
      },
      ["语言符号", "语言", "language"],
      ["α", "β", "γ", "Ω", "Ж", "я", "々", "〆", "〇", "〒"],
    ),
    keywords: ["语言符号", "语言", "language"],
    name: "语言符号",
  },
];

type PanelRow =
  | {
      type: "category";
      id: string;
      title: string;
    }
  | {
      type: "emoji";
      id: string;
      items: any[];
    }
  | {
      type: "kaomoji";
      id: string;
      columns: number;
      items: KaomojiItem[];
    }
  | {
      type: "symbol";
      id: string;
      items: KaomojiItem[];
    };

const EmojiPicker = () => {
  const { rootState } = useContext(MainContext);
  const { t } = useTranslation();
  const scrollerRef = useRef<HTMLDivElement>(null);
  const virtuosoRef = useRef<VirtuosoHandle>(null);
  const pendingCategoryScrollRef = useRef<{
    id: string;
    title: string;
    index: number;
  }>();
  const pendingCategoryUnlockTimeoutRef = useRef<number>();
  const currentVisibleCategoryIdRef = useRef<string>();
  const visibleStartIndexRef = useRef(0);
  const isExpressionGroup =
    rootState.group === "emoji" ||
    rootState.group === "kaomoji" ||
    rootState.group === "symbol";
  const isEmojiGroup = rootState.group === "emoji";
  const isKaomojiGroup = rootState.group === "kaomoji";
  const isSymbolGroup = rootState.group === "symbol";
  const [kaomojiShortColumns, setKaomojiShortColumns] = useState(5);
  const [activeCategoryId, setActiveCategoryId] = useState<
    string | undefined
  >();

  useEffect(() => {
    const container = scrollerRef.current;
    if (!container) return;

    const updateColumns = (width: number) => {
      setKaomojiShortColumns(getShortKaomojiColumns(width));
    };

    updateColumns(container.clientWidth);

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) return;

      updateColumns(entry.contentRect.width);
    });

    observer.observe(container);

    return () => {
      observer.disconnect();
    };
  }, []);

  const listData = useMemo(() => {
    const rows: PanelRow[] = [];
    const keyword = rootState.search?.trim().toLowerCase();

    const appendTextCategories = (
      categories: SymbolCategory[],
      type: "kaomoji" | "symbol",
      perLine: number,
    ) => {
      categories.forEach((category) => {
        const items = category.items.filter((item) => {
          if (!keyword) return true;

          return (
            item.text.toLowerCase().includes(keyword) ||
            category.name.toLowerCase().includes(keyword) ||
            category.keywords.some((entry) =>
              entry.toLowerCase().includes(keyword),
            ) ||
            item.keywords.some((entry) => entry.toLowerCase().includes(keyword))
          );
        });

        if (items.length === 0) return;

        rows.push({
          id: category.id,
          title: t(`clipboard.${type}.categories.${category.id}`),
          type: "category",
        });

        if (type === "kaomoji") {
          chunkKaomojiItems(items, kaomojiShortColumns).forEach(
            (group, index) => {
              rows.push({
                columns: group.columns,
                id: `${category.id}-${index}`,
                items: group.items,
                type,
              });
            },
          );

          return;
        }

        chunk(items, perLine).forEach((group, index) => {
          rows.push({
            id: `${category.id}-${index}`,
            items: group,
            type,
          });
        });
      });
    };

    if (isKaomojiGroup) {
      appendTextCategories(KAOMOJI_CATEGORIES, "kaomoji", 3);

      return rows;
    }

    if (isSymbolGroup) {
      appendTextCategories(SYMBOL_CATEGORIES, "symbol", 6);

      return rows;
    }

    const perLine = 9;

    EMOJI_CATEGORIES.forEach((cat) => {
      if (cat.id === "frequent") return;

      const categoryData = typedData.categories.find((c) => c.id === cat.id);
      if (!categoryData || !categoryData.emojis.length) return;

      const filteredEmojiIds = categoryData.emojis.filter((id) => {
        if (!keyword) return true;
        const emoji = typedData.emojis[id];
        if (!emoji) return false;

        return (
          emoji.name?.toLowerCase().includes(keyword) ||
          cat.name.toLowerCase().includes(keyword) ||
          emoji.keywords?.some((k) => k.toLowerCase().includes(keyword))
        );
      });

      if (filteredEmojiIds.length === 0) return;

      rows.push({
        id: cat.id,
        title: t(`clipboard.emoji.categories.${cat.id}`),
        type: "category",
      });

      chunk(filteredEmojiIds, perLine).forEach((chunkIds, index) => {
        rows.push({
          id: `${cat.id}-${index}`,
          items: chunkIds.map((id) => typedData.emojis[id]),
          type: "emoji",
        });
      });
    });

    return rows;
  }, [isKaomojiGroup, isSymbolGroup, kaomojiShortColumns, rootState.search, t]);

  const expressionTabs = useMemo(() => {
    return listData.reduce<Array<{ id: string; title: string; index: number }>>(
      (result, item, index) => {
        if (item.type === "category") {
          result.push({ id: item.id, index, title: item.title });
        }

        return result;
      },
      [],
    );
  }, [listData]);

  useEffect(() => {
    if (!isExpressionGroup) {
      if (pendingCategoryUnlockTimeoutRef.current) {
        window.clearTimeout(pendingCategoryUnlockTimeoutRef.current);
        pendingCategoryUnlockTimeoutRef.current = void 0;
      }
      pendingCategoryScrollRef.current = void 0;
      setActiveCategoryId(void 0);

      return;
    }

    const firstCategoryId = expressionTabs[0]?.id;
    if (!firstCategoryId) return;

    setActiveCategoryId((current) => {
      return expressionTabs.some((item) => item.id === current)
        ? current
        : firstCategoryId;
    });
  }, [expressionTabs, isExpressionGroup]);

  useEffect(() => {
    if (pendingCategoryUnlockTimeoutRef.current) {
      window.clearTimeout(pendingCategoryUnlockTimeoutRef.current);
      pendingCategoryUnlockTimeoutRef.current = void 0;
    }
    pendingCategoryScrollRef.current = void 0;
  }, [expressionTabs, rootState.group, rootState.search]);

  useEffect(() => {
    return () => {
      if (pendingCategoryUnlockTimeoutRef.current) {
        window.clearTimeout(pendingCategoryUnlockTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!isExpressionGroup || !activeCategoryId) return;

    scrollElementToCenter(
      `expression-tab-${rootState.group}-${activeCategoryId}`,
    );
  }, [activeCategoryId, isExpressionGroup, rootState.group]);

  const handleTextSelect = async (value?: string) => {
    if (!value) return;

    try {
      await pasteTextToClipboard(value, { ignoreHistory: true });
    } catch {
      await copyTextToClipboard(value, { ignoreHistory: true });
    }
  };

  return (
    <div className="emoji-picker-panel mx-3 flex flex-1 flex-col overflow-hidden rounded-md">
      {isExpressionGroup && expressionTabs.length > 0 && (
        <div className="px-2 pb-2">
          <div className="flex overflow-x-auto whitespace-nowrap">
            {expressionTabs.map((item) => {
              const checked = item.id === activeCategoryId;

              return (
                <div
                  id={`expression-tab-${rootState.group}-${item.id}`}
                  key={item.id}
                >
                  <Tag.CheckableTag
                    checked={checked}
                    className={checked ? "bg-primary!" : void 0}
                    onChange={() => {
                      if (item.id === activeCategoryId) return;

                      if (pendingCategoryUnlockTimeoutRef.current) {
                        window.clearTimeout(
                          pendingCategoryUnlockTimeoutRef.current,
                        );
                        pendingCategoryUnlockTimeoutRef.current = void 0;
                      }
                      pendingCategoryScrollRef.current = {
                        id: item.id,
                        index: item.index,
                        title: item.title,
                      };
                      setActiveCategoryId(item.id);
                      virtuosoRef.current?.scrollToIndex({
                        align: "start",
                        behavior: "smooth",
                        index: item.index,
                      });
                    }}
                  >
                    {item.title}
                  </Tag.CheckableTag>
                </div>
              );
            })}
          </div>
        </div>
      )}

      <Scrollbar className="flex-1" offsetX={3} ref={scrollerRef}>
        <Virtuoso
          computeItemKey={(_, item) => item.id}
          customScrollParent={scrollerRef.current ?? void 0}
          data={listData}
          itemContent={(_, data) => {
            if (data.type === "category") {
              return (
                <div className="sticky top-0 z-10 bg-color-1/80 px-3 py-2 font-medium text-color-2 text-sm backdrop-blur-md">
                  {data.title}
                </div>
              );
            }

            if (data.type === "kaomoji") {
              const columns = data.columns;

              return (
                <div
                  className="grid gap-2 px-2 pb-2"
                  style={{
                    gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
                  }}
                >
                  {data.items.map((item) => {
                    return (
                      <button
                        className={`flex h-11 items-center justify-center overflow-hidden rounded-lg bg-transparent px-2 text-center transition-colors hover:bg-color-2 ${getKaomojiItemTextClassName(
                          item.text,
                          columns,
                        )}`}
                        key={item.id}
                        onClick={() => void handleTextSelect(item.text)}
                        title={item.text}
                        type="button"
                      >
                        <span className="block w-full overflow-hidden text-ellipsis whitespace-nowrap">
                          {item.text}
                        </span>
                      </button>
                    );
                  })}
                  {Array.from({ length: columns - data.items.length }).map((_, i) => (
                    <div key={`kaomoji-empty-${i}`} />
                  ))}
                </div>
              );
            }

            if (data.type === "symbol") {
              return (
                <div className="flex gap-2 px-2 pb-2">
                  {data.items.map((item) => {
                    return (
                      <button
                        className="flex h-11 flex-1 items-center justify-center rounded-lg bg-transparent px-1 text-2xl transition-colors hover:bg-color-2"
                        key={item.id}
                        onClick={() => void handleTextSelect(item.text)}
                        title={item.text}
                        type="button"
                      >
                        {item.text}
                      </button>
                    );
                  })}
                  {Array.from({ length: 6 - data.items.length }).map((_, i) => (
                    <div className="flex-1" key={`symbol-empty-${i}`} />
                  ))}
                </div>
              );
            }

            return (
              <div className="flex px-2">
                {data.items?.map((emoji, idx) => {
                  if (!emoji) return <div className="flex-1" key={idx} />;
                  const unified = emoji.skins[0]?.unified;
                  const native = emoji.skins[0]?.native;
                  const imageSrc = getEmojiImageSrc(unified);
                  if (!native) return <div className="flex-1" key={emoji.id} />;

                  return (
                    <button
                      className="flex h-9 flex-1 items-center justify-center rounded-lg bg-transparent transition-colors hover:bg-color-2"
                      key={emoji.id}
                      onClick={() => void handleTextSelect(native)}
                      title={emoji.name}
                      type="button"
                    >
                      {imageSrc ? (
                        <>
                          <img
                            alt={emoji.name}
                            className="h-6 w-6 object-contain"
                            draggable={false}
                            onError={(event) => {
                              event.currentTarget.style.display = "none";
                              const fallback =
                                event.currentTarget.nextElementSibling;
                              if (fallback instanceof HTMLSpanElement) {
                                fallback.style.display = "inline";
                              }
                            }}
                            src={imageSrc}
                          />
                          <span className="hidden text-xl leading-none">
                            {native}
                          </span>
                        </>
                      ) : (
                        native
                      )}
                    </button>
                  );
                })}
                {/* 补充空位保持对齐 */}
                {Array.from({ length: 9 - (data.items?.length || 0) }).map(
                  (_, i) => (
                    <div className="flex-1" key={`empty-${i}`} />
                  ),
                )}
              </div>
            );
          }}
          overscan={isEmojiGroup ? 3000 : 1500}
          rangeChanged={({ startIndex, endIndex }) => {
            if (!isExpressionGroup) return;

            visibleStartIndexRef.current = startIndex;

            const firstVisibleCategory = listData
              .slice(startIndex, endIndex + 1)
              .find((item) => item.type === "category");
            const currentCategory = (() => {
              if (firstVisibleCategory?.type === "category") {
                return firstVisibleCategory;
              }

              for (let index = startIndex; index >= 0; index -= 1) {
                const item = listData[index];

                if (item?.type === "category") {
                  return item;
                }
              }
            })();
            currentVisibleCategoryIdRef.current = currentCategory?.id;

            const pendingCategory = pendingCategoryScrollRef.current;
            if (pendingCategory) {
              const hasReachedTarget = listData
                .slice(startIndex, endIndex + 1)
                .some((item) => {
                  return (
                    item.type === "category" &&
                    (item.id === pendingCategory.id ||
                      item.title === pendingCategory.title)
                  );
                });

              if (hasReachedTarget) {
                if (activeCategoryId !== pendingCategory.id) {
                  setActiveCategoryId(pendingCategory.id);
                }

                if (!pendingCategoryUnlockTimeoutRef.current) {
                  pendingCategoryUnlockTimeoutRef.current = window.setTimeout(
                    () => {
                      pendingCategoryScrollRef.current = void 0;
                      pendingCategoryUnlockTimeoutRef.current = void 0;

                      if (
                        currentVisibleCategoryIdRef.current &&
                        currentVisibleCategoryIdRef.current !== activeCategoryId
                      ) {
                        setActiveCategoryId(
                          currentVisibleCategoryIdRef.current,
                        );
                      }
                    },
                    1000,
                  );
                }
              }

              return;
            }

            if (
              currentCategory?.id &&
              currentCategory.id !== activeCategoryId
            ) {
              setActiveCategoryId(currentCategory.id);
            }
          }}
          ref={virtuosoRef}
        />
      </Scrollbar>
    </div>
  );
};

export default EmojiPicker;
