export const PREVIEW_CACHE_LIMIT = 16;
export const PREVIEW_PANEL_GAP = 40;
export const PREVIEW_PANEL_MARGIN = 32;
export const PREVIEW_PANEL_MIN_HEIGHT = 96;
export const PREVIEW_PANEL_MIN_WIDTH = 288;
export const PREVIEW_PANEL_MAX_HEIGHT = 480;
export const PREVIEW_PANEL_MAX_WIDTH = 480;
export const PREVIEW_PANEL_HEADER_HEIGHT = 48;
export const PREVIEW_PANEL_IMAGE_PADDING_X = 32;
export const PREVIEW_PANEL_IMAGE_PADDING_Y = 32;
export const PREVIEW_EMPTY_CONTENT_HEIGHT = 96;
export const PREVIEW_TEXT_ROW_HEIGHT = 22;
export const PREVIEW_TEXT_VERTICAL_PADDING = 32;
export const PREVIEW_TEXT_SOFT_WRAP_CHARS = 32;
export const PREVIEW_FILE_ROW_HEIGHT = 40;
export const PREVIEW_FILE_VERTICAL_PADDING = 16;
export const PREVIEW_FILE_MORE_FOOTER_HEIGHT = 40;
export const PREVIEW_PANEL_FALLBACK_SIZE = {
  height: 160,
  width: 320,
};
export const PREVIEW_SPRING = {
  damping: 34,
  mass: 0.9,
  stiffness: 420,
};
export const PREVIEW_EXIT_ANIMATION_MS = 160;
export const PREVIEW_PANEL_TRANSITION = {
  duration: 0.18,
  ease: [0.22, 1, 0.36, 1],
} as const;
export const PREVIEW_CONTENT_TRANSITION = {
  duration: 0.12,
  ease: "easeOut",
} as const;
export const PREVIEW_PANEL_VARIANTS = {
  closed: {
    opacity: 0,
    scale: 0.96,
  },
  open: {
    opacity: 1,
    scale: 1,
  },
};
export const PREVIEW_CONNECTOR_VARIANTS = {
  closed: {
    opacity: 0,
  },
  open: {
    opacity: 1,
  },
};
