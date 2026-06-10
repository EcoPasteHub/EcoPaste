import { CAPTURE_KIND_OPTIONS } from "@/constants/captureKinds";
import { ITEM_ACTION_OPTIONS } from "@/constants/itemActions";
import type { PreferenceTab } from "../types/preferences";

const CLICK_ACTION_OPTIONS = [
  { value: "disabled" },
  { value: "singleClickPaste" },
  { value: "doubleClickPaste" },
  { value: "singleClickCopy" },
  { value: "doubleClickCopy" },
];
const MIDDLE_CLICK_ACTION_OPTIONS = [
  { value: "disabled" },
  { value: "singleClickPaste" },
  { value: "singleClickPastePlain" },
  { value: "singleClickCopy" },
  { value: "singleClickCopyPlain" },
];
const CLIPBOARD_SORT_OPTIONS = [
  { value: "createdAtDesc" },
  { value: "updatedAtDesc" },
  { value: "useCountDesc" },
];
export const preferenceTabs: PreferenceTab[] = [
  {
    icon: "i-lucide:clipboard-plus",
    id: "record",
    sections: [
      {
        id: "capture",
        settings: [
          {
            control: { type: "switch" },
            id: "capture.text",
            keywords: ["text", "plain", "record"],
            path: ["clipboard", "capture", "text"],
            value: (settings) => {
              return settings.clipboard.capture.text;
            },
          },
          {
            control: { type: "switch" },
            id: "capture.html",
            keywords: ["html", "rich text", "format"],
            path: ["clipboard", "capture", "html"],
            value: (settings) => {
              return settings.clipboard.capture.html;
            },
          },
          {
            control: { type: "switch" },
            id: "capture.rtf",
            keywords: ["rtf", "rich text", "format"],
            path: ["clipboard", "capture", "rtf"],
            value: (settings) => {
              return settings.clipboard.capture.rtf;
            },
          },
          {
            control: { type: "switch" },
            id: "capture.image",
            keywords: ["image", "picture", "thumbnail"],
            path: ["clipboard", "capture", "image"],
            value: (settings) => {
              return settings.clipboard.capture.image;
            },
          },
          {
            control: { type: "switch" },
            id: "capture.files",
            keywords: ["file", "folder", "path"],
            path: ["clipboard", "capture", "files"],
            value: (settings) => {
              return settings.clipboard.capture.files;
            },
          },
          {
            control: { min: 0, suffixKey: "mb", type: "number" },
            id: "capture.maxTextMb",
            keywords: ["text", "size", "limit", "mb"],
            path: ["clipboard", "capture", "maxTextMb"],
            value: (settings) => {
              return settings.clipboard.capture.maxTextMb;
            },
          },
          {
            control: { min: 0, suffixKey: "mb", type: "number" },
            id: "capture.maxImageMb",
            keywords: ["image", "picture", "size", "limit", "mb"],
            path: ["clipboard", "capture", "maxImageMb"],
            value: (settings) => {
              return settings.clipboard.capture.maxImageMb;
            },
          },
          {
            control: {
              options: CAPTURE_KIND_OPTIONS,
              type: "sortableTree",
            },
            id: "capture.order",
            keywords: ["priority", "order", "format", "rich text"],
            path: ["clipboard", "capture", "order"],
            value: (settings) => {
              return settings.clipboard.capture.order;
            },
          },
        ],
      },
      {
        id: "source",
        settings: [
          {
            control: { type: "appExclusion" },
            id: "source.excludedApps",
            keywords: ["exclude", "ignore", "app", "source"],
            path: ["clipboard", "filters", "excludedAppIds"],
            value: (settings) => {
              return settings.clipboard.filters.excludedAppIds;
            },
          },
        ],
      },
      {
        id: "sensitive",
        settings: [
          {
            control: { type: "switch" },
            id: "sensitive.collectSecrets",
            keywords: ["token", "key", "secret", "code"],
            path: ["clipboard", "sensitive", "collectSecrets"],
            value: (settings) => {
              return settings.clipboard.sensitive.collectSecrets;
            },
          },
          {
            control: { type: "switch" },
            id: "sensitive.redactSecrets",
            keywords: ["token", "key", "secret", "redact", "mask"],
            path: ["clipboard", "sensitive", "redactSecrets"],
            value: (settings) => {
              return settings.clipboard.sensitive.redactSecrets;
            },
          },
        ],
      },
    ],
  },
  {
    icon: "i-lucide:history",
    id: "organize",
    sections: [
      {
        id: "history",
        settings: [
          {
            control: { type: "retention" },
            id: "history.retention",
            keywords: ["retention", "cleanup", "history"],
            path: ["clipboard", "history", "retention"],
            value: (settings) => {
              return settings.clipboard.history.retention;
            },
          },
          {
            control: { min: 0, suffixKey: "items", type: "number" },
            id: "history.maxCount",
            keywords: ["max", "count", "limit"],
            path: ["clipboard", "history", "maxCount"],
            value: (settings) => {
              return settings.clipboard.history.maxCount;
            },
          },
          {
            control: { min: 0, suffixKey: "hours", type: "number" },
            id: "history.cleanupIntervalHours",
            keywords: ["cleanup", "interval", "schedule"],
            path: ["clipboard", "history", "cleanupIntervalHours"],
            value: (settings) => {
              return settings.clipboard.history.cleanupIntervalHours;
            },
          },
        ],
      },
      {
        id: "organizing",
        settings: [
          {
            control: { type: "switch" },
            id: "organizing.autoFavorite",
            keywords: ["note", "favorite", "auto"],
            path: ["clipboard", "content", "autoFavorite"],
            value: (settings) => {
              return settings.clipboard.content.autoFavorite;
            },
          },
        ],
      },
      {
        id: "groups",
        settings: [
          {
            control: { type: "action" },
            id: "organizing.customGroups",
            keywords: ["group", "folder", "organize"],
          },
        ],
      },
      {
        id: "search",
        settings: [
          {
            control: { type: "switch" },
            id: "search.defaultFocus",
            keywords: ["search", "focus", "open"],
            path: ["clipboard", "search", "defaultFocus"],
            value: (settings) => {
              return settings.clipboard.search.defaultFocus;
            },
          },
          {
            control: { type: "switch" },
            id: "search.clearOnHide",
            keywords: ["search", "clear", "hide"],
            path: ["clipboard", "search", "clearOnHide"],
            value: (settings) => {
              return settings.clipboard.search.clearOnHide;
            },
          },
          {
            control: { options: CLIPBOARD_SORT_OPTIONS, type: "select" },
            id: "search.sort",
            keywords: ["sort", "frequency", "usage", "created", "updated"],
            path: ["clipboard", "content", "sort"],
            value: (settings) => {
              return settings.clipboard.content.sort;
            },
          },
        ],
      },
    ],
  },
  {
    icon: "i-lucide:mouse-pointer-click",
    id: "reuse",
    sections: [
      {
        id: "paste",
        settings: [
          {
            control: {
              options: CLICK_ACTION_OPTIONS,
              type: "segmented",
            },
            id: "paste.autoPaste",
            keywords: ["paste", "click", "auto"],
            path: ["clipboard", "content", "autoPaste"],
            value: (settings) => {
              return settings.clipboard.content.autoPaste;
            },
          },
          {
            control: {
              options: MIDDLE_CLICK_ACTION_OPTIONS,
              type: "segmented",
            },
            id: "paste.middleClick",
            keywords: ["paste", "middle click", "mouse"],
            path: ["clipboard", "content", "middleClick"],
            value: (settings) => {
              return settings.clipboard.content.middleClick;
            },
          },
          {
            control: { type: "switch" },
            id: "paste.plainDefault",
            keywords: ["plain", "paste", "format"],
            path: ["clipboard", "content", "pastePlain"],
            value: (settings) => {
              return settings.clipboard.content.pastePlain;
            },
          },
          {
            control: { type: "switch" },
            id: "paste.fileMode",
            keywords: ["file", "path", "paste"],
            path: ["clipboard", "content", "pasteFilesAsPath"],
            value: (settings) => {
              return settings.clipboard.content.pasteFilesAsPath;
            },
          },
        ],
      },
      {
        id: "copy",
        settings: [
          {
            control: { type: "switch" },
            id: "copy.plainDefault",
            keywords: ["copy", "plain", "format"],
            path: ["clipboard", "content", "copyPlain"],
            value: (settings) => {
              return settings.clipboard.content.copyPlain;
            },
          },
          {
            control: { type: "switch" },
            id: "copy.hideWindow",
            keywords: ["copy", "hide", "window"],
            path: ["clipboard", "content", "copyThenHideWindow"],
            value: (settings) => {
              return settings.clipboard.content.copyThenHideWindow;
            },
          },
          {
            control: { type: "switch" },
            id: "copy.updateOnReuse",
            keywords: ["copy", "paste", "reuse", "sort", "frequency"],
            path: ["clipboard", "content", "updateOnReuse"],
            value: (settings) => {
              return settings.clipboard.content.updateOnReuse;
            },
          },
          {
            control: { type: "switch" },
            id: "copy.sound",
            keywords: ["sound", "feedback", "copy"],
            path: ["clipboard", "feedback", "copySound"],
            value: (settings) => {
              return settings.clipboard.feedback.copySound;
            },
          },
        ],
      },
      {
        id: "actions",
        settings: [
          {
            control: {
              options: ITEM_ACTION_OPTIONS,
              orderPath: ["clipboard", "content", "itemActionOrder"],
              type: "sortableCheckboxTree",
            },
            id: "actions.visible",
            keywords: ["action", "hover", "buttons"],
            path: ["clipboard", "content", "itemActions"],
            value: (settings) => {
              return {
                order: settings.clipboard.content.itemActionOrder,
                selected: settings.clipboard.content.itemActions,
              };
            },
          },
          {
            control: { type: "switch" },
            id: "actions.deleteConfirm",
            keywords: ["delete", "confirm"],
            path: ["clipboard", "content", "deleteConfirm"],
            value: (settings) => {
              return settings.clipboard.content.deleteConfirm;
            },
          },
          {
            control: { type: "switch" },
            id: "actions.deleteFavoriteConfirm",
            keywords: ["delete", "favorite", "confirm"],
            path: ["clipboard", "content", "deleteFavoriteConfirm"],
            value: (settings) => {
              return settings.clipboard.content.deleteFavoriteConfirm;
            },
          },
          {
            control: { type: "switch" },
            id: "actions.deleteFavoriteItems",
            keywords: ["delete", "favorite", "allow"],
            path: ["clipboard", "content", "deleteFavoriteItems"],
            value: (settings) => {
              return settings.clipboard.content.deleteFavoriteItems;
            },
          },
          {
            control: { type: "switch" },
            id: "actions.deletePinnedConfirm",
            keywords: ["delete", "pinned", "pin", "confirm"],
            path: ["clipboard", "content", "deletePinnedConfirm"],
            value: (settings) => {
              return settings.clipboard.content.deletePinnedConfirm;
            },
          },
          {
            control: { type: "switch" },
            id: "actions.deletePinnedItems",
            keywords: ["delete", "pinned", "pin", "allow"],
            path: ["clipboard", "content", "deletePinnedItems"],
            value: (settings) => {
              return settings.clipboard.content.deletePinnedItems;
            },
          },
          {
            control: { type: "switch" },
            id: "actions.deleteFavoriteItemsOnlyInFavoriteGroup",
            keywords: ["delete", "favorite", "group"],
            path: [
              "clipboard",
              "content",
              "deleteFavoriteItemsOnlyInFavoriteGroup",
            ],
            value: (settings) => {
              return settings.clipboard.content
                .deleteFavoriteItemsOnlyInFavoriteGroup;
            },
          },
        ],
      },
    ],
  },
  {
    icon: "i-lucide:panel-top",
    id: "workflow",
    sections: [
      {
        id: "window",
        settings: [
          {
            control: {
              options: [
                { value: "followCursor" },
                { value: "center" },
                { value: "remember" },
              ],
              type: "segmented",
            },
            id: "window.position",
            keywords: ["window", "position", "cursor"],
            path: ["clipboard", "window", "position"],
            value: (settings) => {
              return settings.clipboard.window.position;
            },
          },
          {
            control: { type: "switch" },
            id: "window.scrollToTopOnOpen",
            keywords: ["window", "scroll", "top", "open"],
            path: ["clipboard", "window", "scrollToTopOnOpen"],
            value: (settings) => {
              return settings.clipboard.window.scrollToTopOnOpen;
            },
          },
          {
            control: { type: "switch" },
            id: "window.selectAllGroupOnOpen",
            keywords: ["window", "group", "all", "open"],
            path: ["clipboard", "window", "selectAllGroupOnOpen"],
            value: (settings) => {
              return settings.clipboard.window.selectAllGroupOnOpen;
            },
          },
        ],
      },
      {
        id: "preview",
        settings: [
          {
            control: { type: "switch" },
            id: "preview.hover",
            keywords: ["preview", "hover"],
            path: ["clipboard", "preview", "hoverEnabled"],
            value: (settings) => {
              return settings.clipboard.preview.hoverEnabled;
            },
          },
          {
            control: {
              options: [
                { value: "ms300" },
                { value: "ms500" },
                { value: "ms1000" },
              ],
              type: "segmented",
            },
            disabledWhen: (settings) => {
              return !settings.clipboard.preview.hoverEnabled;
            },
            id: "preview.delay",
            keywords: ["preview", "delay", "hover"],
            parentId: "preview.hover",
            path: ["clipboard", "preview", "hoverDelayMs"],
            value: (settings) => {
              return settings.clipboard.preview.hoverDelayMs;
            },
          },
          {
            control: { type: "switch" },
            id: "preview.space",
            keywords: ["space", "preview", "keyboard"],
            path: ["clipboard", "preview", "spaceEnabled"],
            value: (settings) => {
              return settings.clipboard.preview.spaceEnabled;
            },
          },
        ],
      },
      {
        id: "appearance",
        settings: [
          {
            control: {
              options: [
                { value: "auto" },
                { value: "light" },
                { value: "dark" },
              ],
              type: "segmented",
            },
            id: "appearance.theme",
            keywords: ["theme", "dark", "light"],
            path: ["appearance", "theme"],
            value: (settings) => {
              return settings.appearance.theme;
            },
          },
          {
            control: {
              options: [{ value: "zh-CN" }, { value: "en-US" }],
              type: "segmented",
            },
            id: "appearance.language",
            keywords: ["language", "locale", "english"],
            path: ["appearance", "language"],
            value: (settings) => {
              return settings.appearance.language;
            },
          },
          {
            control: { max: 5, min: 1, suffixKey: "lines", type: "number" },
            id: "appearance.textMaxLines",
            keywords: ["density", "text", "line", "compact"],
            path: ["clipboard", "display", "textMaxLines"],
            value: (settings) => {
              return settings.clipboard.display.textMaxLines;
            },
          },
          {
            control: { max: 100, min: 20, suffixKey: "px", type: "number" },
            id: "appearance.imageMaxHeight",
            keywords: ["density", "image", "height", "thumbnail"],
            path: ["clipboard", "display", "imageMaxHeight"],
            value: (settings) => {
              return settings.clipboard.display.imageMaxHeight;
            },
          },
          {
            control: { max: 5, min: 1, suffixKey: "files", type: "number" },
            id: "appearance.fileMaxCount",
            keywords: ["density", "file", "count", "array"],
            path: ["clipboard", "display", "fileMaxCount"],
            value: (settings) => {
              return settings.clipboard.display.fileMaxCount;
            },
          },
          {
            control: { type: "switch" },
            id: "appearance.showOriginalPreview",
            keywords: ["note", "hover", "original", "preview"],
            path: ["clipboard", "content", "showOriginalPreview"],
            value: (settings) => {
              return settings.clipboard.content.showOriginalPreview;
            },
          },
        ],
      },
      {
        id: "control",
        settings: [
          {
            control: { type: "switch" },
            id: "control.autoStart",
            keywords: ["startup", "login", "autostart"],
            path: ["general", "autoStart"],
            value: (settings) => {
              return settings.general.autoStart;
            },
          },
          {
            control: { type: "switch" },
            disabledWhen: (settings) => {
              return !settings.general.autoStart;
            },
            id: "control.silentStart",
            keywords: ["startup", "silent", "login"],
            parentId: "control.autoStart",
            path: ["general", "silentStart"],
            value: (settings) => {
              return settings.general.silentStart;
            },
          },
          {
            control: { type: "switch" },
            id: "control.trayIcon",
            keywords: ["tray", "menu bar", "system"],
            path: ["general", "trayIcon"],
            value: (settings) => {
              return settings.general.trayIcon;
            },
          },
          {
            control: { type: "switch" },
            id: "control.dockIcon",
            keywords: ["dock", "taskbar", "icon"],
            path: ["general", "dockIcon"],
            value: (settings) => {
              return settings.general.dockIcon;
            },
          },
        ],
      },
    ],
  },
  {
    icon: "i-lucide:keyboard",
    id: "shortcuts",
    sections: [
      {
        id: "globalShortcuts",
        settings: [
          {
            control: { type: "shortcutRecorder" },
            id: "shortcuts.openClipboard",
            keywords: ["shortcut", "hotkey", "open"],
            path: ["shortcuts", "openClipboard"],
            value: (settings) => {
              return settings.shortcuts.openClipboard;
            },
          },
          {
            control: { type: "shortcutRecorder" },
            id: "shortcuts.openPreference",
            keywords: ["shortcut", "hotkey", "preference"],
            path: ["shortcuts", "openPreference"],
            value: (settings) => {
              return settings.shortcuts.openPreference;
            },
          },
        ],
      },
    ],
  },
  {
    icon: "i-lucide:database",
    id: "data",
    sections: [
      {
        id: "localData",
        settings: [
          {
            control: { type: "action" },
            id: "localData.dataDirectory",
            keywords: ["database", "sqlite", "local", "cache", "image", "icon"],
          },
          {
            control: { type: "action" },
            id: "localData.logDirectory",
            keywords: ["log", "diagnostic"],
          },
          {
            control: { type: "action" },
            id: "localData.cleanCache",
            keywords: ["cache", "clean", "storage"],
          },
        ],
      },
      {
        id: "backup",
        settings: [
          {
            control: { type: "action" },
            id: "backup.exportHistory",
            keywords: ["export", "backup", "history"],
          },
          {
            control: { type: "action" },
            id: "backup.importHistory",
            keywords: ["import", "backup", "history"],
          },
        ],
      },
      {
        id: "diagnostics",
        settings: [
          {
            control: { danger: true, type: "action" },
            id: "diagnostics.resetPreferences",
            keywords: ["reset", "preferences"],
          },
        ],
      },
      {
        id: "external",
        settings: [
          {
            control: { type: "action" },
            disabled: true,
            id: "external.cli",
            keywords: ["cli", "terminal", "json"],
            status: "comingSoon",
          },
        ],
      },
      {
        id: "updates",
        settings: [
          {
            control: { type: "switch" },
            id: "updates.autoCheck",
            keywords: ["update", "version"],
            path: ["update", "autoCheck"],
            value: (settings) => {
              return settings.update.autoCheck;
            },
          },
          {
            control: { type: "switch" },
            disabledWhen: (settings) => {
              return !settings.update.autoCheck;
            },
            id: "updates.beta",
            keywords: ["beta", "update"],
            parentId: "updates.autoCheck",
            path: ["update", "includeBeta"],
            value: (settings) => {
              return settings.update.includeBeta;
            },
          },
          {
            control: {
              options: [
                { value: "daily" },
                { value: "weekly" },
                { value: "monthly" },
              ],
              type: "segmented",
            },
            disabledWhen: (settings) => {
              return !settings.update.autoCheck;
            },
            id: "updates.frequency",
            keywords: ["update", "frequency", "schedule"],
            parentId: "updates.autoCheck",
            path: ["update", "frequency"],
            value: (settings) => {
              return settings.update.frequency;
            },
          },
        ],
      },
    ],
  },
  {
    icon: "i-lucide:info",
    id: "about",
    sections: [],
  },
];

export const allPreferenceSettings = preferenceTabs.flatMap((tab) => {
  return tab.sections.flatMap((section) => {
    return section.settings.map((setting) => {
      return { section, setting, tab };
    });
  });
});
