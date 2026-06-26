/* @unocss-include */
interface SettingVisual {
  tone: string;
  icon: string;
}

const SETTING_ICON_MAP: Record<string, string> = {
  "about.checkUpdates": "i-lucide:refresh-cw",
  "about.github": "i-lucide:github",
  "about.sponsor": "i-ph:hand-heart",
  "actions.deleteConfirm": "i-lucide:trash-2",
  "actions.deleteFavoriteConfirm": "i-lucide:shield-alert",
  "actions.deleteFavoriteItems": "i-lucide:star-off",
  "actions.deleteFavoriteItemsOnlyInFavoriteGroup": "i-lucide:shield-check",
  "actions.deletePinnedConfirm": "i-lucide:shield-alert",
  "actions.deletePinnedItems": "i-ph:push-pin-slash-bold",
  "actions.visible": "i-lucide:square-mouse-pointer",
  "appearance.fileMaxCount": "i-lucide:files",
  "appearance.imageMaxHeight": "i-lucide:image",
  "appearance.language": "i-lucide:languages",
  "appearance.showOriginalPreview": "i-lucide:mouse-pointer-2",
  "appearance.textMaxLines": "i-lucide:rows-3",
  "appearance.theme": "i-lucide:paintbrush-vertical",
  "backup.exportHistory": "i-lucide:upload",
  "backup.importHistory": "i-lucide:download",
  "capture.files": "i-lucide:files",
  "capture.html": "i-lucide:file-code-2",
  "capture.image": "i-lucide:file-image",
  "capture.maxImageMb": "i-lucide:image-up",
  "capture.maxTextMb": "i-lucide:letter-text",
  "capture.order": "i-lucide:list-ordered",
  "capture.rtf": "i-lucide:file-type",
  "capture.text": "i-lucide:clipboard-type",
  "control.autoStart": "i-lucide:log-in",
  "control.dockIcon": "i-lucide:panel-bottom",
  "control.silentStart": "i-lucide:eye-off",
  "control.trayIcon": "i-lucide:panel-top",
  "copy.hideWindow": "i-lucide:panel-top-close",
  "copy.plainDefault": "i-lucide:clipboard-copy",
  "copy.sound": "i-lucide:volume-2",
  "copy.updateOnReuse": "i-lucide:refresh-cw",
  "diagnostics.resetPreferences": "i-lucide:settings-2",
  "diagnostics.windowLifecycle": "i-lucide:activity",
  "history.cleanupIntervalHours": "i-lucide:timer-reset",
  "history.maxCount": "i-lucide:list-ordered",
  "history.retention": "i-lucide:calendar-clock",
  "localData.cleanCache": "i-lucide:brush-cleaning",
  "localData.dataDirectory": "i-lucide:folder-root",
  "localData.logDirectory": "i-lucide:folder-open-dot",
  "organizing.autoFavorite": "i-lucide:message-square-heart",
  "organizing.customGroups": "i-lucide:folder-tree",
  "paste.autoPaste": "i-ph:mouse-left-click-bold",
  "paste.fileMode": "i-lucide:file-symlink",
  "paste.middleClick": "i-ph:mouse-middle-click-bold",
  "paste.plainDefault": "i-lucide:clipboard-type",
  "preview.delay": "i-lucide:timer",
  "preview.hover": "i-lucide:mouse-pointer-2",
  "preview.space": "i-lucide:space",
  "search.clearOnHide": "i-lucide:eraser",
  "search.defaultFocus": "i-lucide:scan-search",
  "search.sort": "i-lucide:arrow-down-up",
  "sensitive.collectSecrets": "i-lucide:key-round",
  "sensitive.redactSecrets": "i-lucide:scan-eye",
  "shortcuts.openClipboard": "i-lucide:app-window",
  "shortcuts.openPreference": "i-lucide:settings",
  "shortcuts.winV": "i-lucide:clipboard-list",
  "source.excludedApps": "i-lucide:shield-ban",
  "updates.autoCheck": "i-lucide:refresh-cw",
  "updates.beta": "i-lucide:flask-conical",
  "updates.frequency": "i-lucide:calendar-sync",
  "window.idleDestroySeconds": "i-lucide:timer-reset",
  "window.lightweightMode": "i-lucide:leaf",
  "window.position": "i-lucide:move",
  "window.scrollToTopOnOpen": "i-lucide:arrow-up-to-line",
  "window.selectCategoryOnOpen": "i-lucide:shapes",
  "window.selectGroupOnOpen": "i-lucide:folder-tree",
  "window.selectRangeOnOpen": "i-lucide:list-filter",
};

/**
 * 根据设置 id 选择图标与色调，保持 schema 只描述信息架构。
 */
export function resolveSettingVisual(id: string): SettingVisual {
  return settingVisual(SETTING_ICON_MAP[id] ?? "i-lucide:circle");
}

/**
 * 统一设置行图标质感，避免行内语义色造成视觉噪音。
 */
function settingVisual(icon: string): SettingVisual {
  return {
    icon,
    tone: "text-ant-tertiary group-hover:text-ant-secondary",
  };
}
