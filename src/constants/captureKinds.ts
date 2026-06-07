/* @unocss-include */
import type { TFunction } from "i18next";
import type { CaptureKind } from "@/types/settings";

type PreferenceTranslator = TFunction<"preferences">;

interface CaptureKindMeta {
  icon: string;
  labelKey: string;
}

export const CAPTURE_KIND_ORDER: CaptureKind[] = [
  "files",
  "image",
  "html",
  "rtf",
  "text",
];

const CAPTURE_KIND_META: Record<CaptureKind, CaptureKindMeta> = {
  files: {
    icon: "i-lucide:files",
    labelKey: "schema.captureKinds.files",
  },
  html: {
    icon: "i-lucide:file-code-2",
    labelKey: "schema.captureKinds.html",
  },
  image: {
    icon: "i-lucide:file-image",
    labelKey: "schema.captureKinds.image",
  },
  rtf: {
    icon: "i-lucide:file-type",
    labelKey: "schema.captureKinds.rtf",
  },
  text: {
    icon: "i-lucide:clipboard-type",
    labelKey: "schema.captureKinds.text",
  },
};

export const CAPTURE_KIND_OPTIONS = CAPTURE_KIND_ORDER.map((kind) => {
  return { value: kind };
});

const CAPTURE_KIND_KEYS = new Set<CaptureKind>(CAPTURE_KIND_ORDER);

/**
 * 判断未知字符串是否为剪贴板采集类型。
 */
export function isCaptureKind(value: string): value is CaptureKind {
  return CAPTURE_KIND_KEYS.has(value as CaptureKind);
}

/**
 * 返回采集类型在偏好设置树中的图标类名。
 */
export function resolveCaptureKindIcon(kind: CaptureKind) {
  return CAPTURE_KIND_META[kind].icon;
}

/**
 * 翻译采集类型展示文案。
 */
export function translateCaptureKindLabel(
  t: PreferenceTranslator,
  kind: CaptureKind,
) {
  const meta = CAPTURE_KIND_META[kind];

  return t(meta.labelKey);
}
