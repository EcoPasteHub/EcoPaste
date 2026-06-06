/* @unocss-include */
interface SettingVisual {
  tone: string;
  icon: string;
}

const SETTING_ICON_MAP: Record<string, string> = {
  "actions.contextual": "i-lucide:wand-sparkles",
  "actions.deleteConfirm": "i-lucide:trash-2",
  "actions.transforms": "i-ph:magic-wand",
  "actions.visible": "i-lucide:square-mouse-pointer",
  "appearance.fileMaxCount": "i-lucide:files",
  "appearance.imageMaxHeight": "i-lucide:image",
  "appearance.language": "i-lucide:languages",
  "appearance.reducedMotion": "i-lucide:accessibility",
  "appearance.textMaxLines": "i-lucide:rows-3",
  "appearance.theme": "i-lucide:paintbrush-vertical",
  "backup.exportHistory": "i-lucide:upload",
  "backup.importHistory": "i-lucide:download",
  "capture.files": "i-lucide:files",
  "capture.html": "i-lucide:file-code-2",
  "capture.image": "i-lucide:file-image",
  "capture.rtf": "i-lucide:file-type",
  "capture.text": "i-lucide:clipboard-type",
  "control.autoStart": "i-lucide:log-in",
  "control.dockIcon": "i-lucide:panel-bottom",
  "control.silentStart": "i-lucide:eye-off",
  "control.trayIcon": "i-lucide:panel-top",
  "copy.plainDefault": "i-lucide:copy-slash",
  "copy.sound": "i-ph:speaker-high",
  "copy.updateOnReuse": "i-lucide:refresh-cw",
  "diagnostics.resetPreferences": "i-lucide:settings-2",
  "diagnostics.resetWindow": "i-lucide:panel-top-open",
  "external.ai": "i-lucide:brain-circuit",
  "external.cli": "i-lucide:square-terminal",
  "external.mcp": "i-lucide:plug-zap",
  "history.keepFavorite": "i-lucide:star",
  "history.keepPinned": "i-lucide:pin",
  "history.maxCount": "i-lucide:list-ordered",
  "history.maxSize": "i-lucide:gauge",
  "history.retention": "i-lucide:calendar-clock",
  "localData.cleanCache": "i-lucide:eraser",
  "localData.dataDirectory": "i-lucide:folder-root",
  "localData.logDirectory": "i-lucide:folder-open-dot",
  "organizing.autoFavorite": "i-lucide:message-square-heart",
  "organizing.customGroups": "i-lucide:folder-tree",
  "paste.autoPaste": "i-ph:mouse-left-click",
  "paste.fileMode": "i-lucide:file-input",
  "paste.middleClick": "i-ph:mouse-middle-click",
  "paste.plainDefault": "i-lucide:clipboard-type",
  "preview.delay": "i-lucide:timer",
  "preview.hover": "i-lucide:mouse-pointer-2",
  "preview.original": "i-lucide:file-search",
  "preview.space": "i-lucide:space",
  "search.clearOnHide": "i-lucide:eraser",
  "search.defaultFocus": "i-lucide:scan-search",
  "search.sort": "i-lucide:arrow-down-up",
  "sensitive.keywordRules": "i-lucide:list-filter-plus",
  "sensitive.secretDetection": "i-lucide:key-round",
  "shortcuts.openClipboard": "i-lucide:panel-top-open",
  "shortcuts.openPreference": "i-lucide:settings",
  "source.excludedApps": "i-lucide:shield-ban",
  "source.scanDirs": "i-lucide:folder-search",
  "updates.autoCheck": "i-lucide:refresh-cw",
  "updates.beta": "i-lucide:flask-conical",
  "updates.frequency": "i-lucide:calendar-sync",
  "window.position": "i-lucide:move",
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
