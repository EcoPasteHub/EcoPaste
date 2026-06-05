interface SettingVisual {
  tone: string;
  icon: string;
}

const SETTING_ICON_MAP: Record<string, string> = {
  "actions.contextual": "i-lucide:wand-sparkles",
  "actions.deleteConfirm": "i-lucide:trash-2",
  "actions.transforms": "i-ph:magic-wand",
  "actions.visible": "i-lucide:square-mouse-pointer",
  "appearance.density": "i-lucide:rows-3",
  "appearance.language": "i-lucide:languages",
  "appearance.reducedMotion": "i-lucide:accessibility",
  "appearance.theme": "i-lucide:paintbrush-vertical",
  "backup.exportHistory": "i-lucide:upload",
  "backup.importHistory": "i-lucide:download",
  "capture.files": "i-lucide:files",
  "capture.html": "i-lucide:file-code-2",
  "capture.images": "i-lucide:file-image",
  "capture.rtf": "i-lucide:file-type",
  "capture.text": "i-lucide:clipboard-type",
  "control.autoStart": "i-lucide:rocket",
  "control.dockIcon": "i-lucide:dock",
  "control.silentStart": "i-lucide:moon",
  "control.trayIcon": "i-ph:tray",
  "copy.plainDefault": "i-lucide:copy-slash",
  "copy.sound": "i-ph:speaker-high",
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
  "localData.cleanCache": "i-lucide:folder-x",
  "localData.dataDirectory": "i-lucide:hard-drive",
  "localData.logDirectory": "i-lucide:logs",
  "organizing.autoFavorite": "i-lucide:badge-plus",
  "organizing.customGroups": "i-lucide:folder-tree",
  "organizing.favorite": "i-lucide:star",
  "organizing.notes": "i-lucide:notebook-pen",
  "organizing.pinned": "i-lucide:pin",
  "paste.autoPaste": "i-ph:mouse-left-click",
  "paste.fileMode": "i-lucide:file-input",
  "paste.middleClick": "i-ph:mouse-middle-click",
  "paste.plainDefault": "i-lucide:clipboard-type",
  "paste.secondaryClick": "i-ph:mouse-right-click",
  "preview.delay": "i-lucide:timer",
  "preview.hover": "i-lucide:mouse-pointer-2",
  "preview.original": "i-lucide:file-search",
  "preview.space": "i-lucide:space",
  "search.clearOnHide": "i-lucide:eraser",
  "search.defaultFocus": "i-lucide:scan-search",
  "search.frequencySort": "i-lucide:arrow-down-up",
  "sensitive.keywordRules": "i-lucide:list-filter-plus",
  "sensitive.passwordApps": "i-lucide:lock-keyhole",
  "sensitive.privateMode": "i-lucide:pause",
  "sensitive.secretDetection": "i-lucide:key-round",
  "shortcuts.openClipboard": "i-lucide:panel-top-open",
  "shortcuts.openPreference": "i-lucide:settings",
  "shortcuts.windowDelete": "i-ph:backspace",
  "shortcuts.windowFavorite": "i-lucide:star",
  "shortcuts.windowFocusSearch": "i-lucide:search",
  "shortcuts.windowNavigate": "i-lucide:arrow-up-down",
  "shortcuts.windowOpenPreference": "i-lucide:settings",
  "shortcuts.windowPaste": "i-ph:key-return",
  "shortcuts.windowPastePlain": "i-lucide:clipboard-type",
  "shortcuts.windowPin": "i-lucide:pin",
  "shortcuts.windowQuickPaste": "i-lucide:list-start",
  "source.detectApp": "i-lucide:app-window",
  "source.excludedApps": "i-lucide:shield-ban",
  "source.perAppRules": "i-lucide:sliders-horizontal",
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
