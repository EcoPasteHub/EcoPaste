import { CAPTURE_KIND_OPTIONS } from "@/constants/captureKinds";
import { ITEM_ACTION_OPTIONS } from "@/constants/itemActions";
import type { PreferenceTab } from "../types/preferences";

const CLICK_ACTION_OPTIONS = [
  { label: "禁用", value: "disabled" },
  { label: "单击粘贴", value: "singleClickPaste" },
  { label: "双击粘贴", value: "doubleClickPaste" },
  { label: "单击复制", value: "singleClickCopy" },
  { label: "双击复制", value: "doubleClickCopy" },
];
const MIDDLE_CLICK_ACTION_OPTIONS = [
  { label: "禁用", value: "disabled" },
  { label: "粘贴", value: "singleClickPaste" },
  { label: "粘贴为纯文本", value: "singleClickPastePlain" },
  { label: "复制", value: "singleClickCopy" },
  { label: "复制为纯文本", value: "singleClickCopyPlain" },
];
const CLIPBOARD_SORT_OPTIONS = [
  { label: "创建时间", value: "createdAtDesc" },
  { label: "更新时间", value: "updatedAtDesc" },
  { label: "使用频率", value: "useCountDesc" },
];

export const preferenceTabs: PreferenceTab[] = [
  {
    icon: "i-lucide:clipboard-plus",
    id: "record",
    sections: [
      {
        description: "选择哪些剪贴板内容会保存到历史。",
        id: "capture",
        settings: [
          {
            control: { type: "switch" },
            description: "关闭后，复制的纯文本不会进入历史。",
            id: "capture.text",
            keywords: ["text", "plain", "record"],
            path: ["clipboard", "capture", "text"],
            title: "纯文本",
            value: (settings) => {
              return settings.clipboard.capture.text;
            },
          },
          {
            control: { type: "switch" },
            description: "关闭后，带 HTML 格式的网页富文本不会进入历史。",
            id: "capture.html",
            keywords: ["html", "rich text", "format"],
            path: ["clipboard", "capture", "html"],
            title: "HTML 内容",
            value: (settings) => {
              return settings.clipboard.capture.html;
            },
          },
          {
            control: { type: "switch" },
            description: "关闭后，带 RTF 格式的文档富文本不会进入历史。",
            id: "capture.rtf",
            keywords: ["rtf", "rich text", "format"],
            path: ["clipboard", "capture", "rtf"],
            title: "RTF 内容",
            value: (settings) => {
              return settings.clipboard.capture.rtf;
            },
          },
          {
            control: { type: "switch" },
            description: "关闭后，剪贴板中的图片不会进入历史。",
            id: "capture.image",
            keywords: ["image", "picture", "thumbnail"],
            path: ["clipboard", "capture", "image"],
            title: "图片",
            value: (settings) => {
              return settings.clipboard.capture.image;
            },
          },
          {
            control: { type: "switch" },
            description: "关闭后，复制的文件和文件夹不会进入历史。",
            id: "capture.files",
            keywords: ["file", "folder", "path"],
            path: ["clipboard", "capture", "files"],
            title: "文件和文件夹",
            value: (settings) => {
              return settings.clipboard.capture.files;
            },
          },
          {
            control: {
              label: "管理",
              options: CAPTURE_KIND_OPTIONS,
              type: "sortableTree",
            },
            description:
              "剪贴板同时包含多种格式时，按这个顺序选择要保存的内容。",
            id: "capture.order",
            keywords: ["priority", "order", "format", "rich text"],
            path: ["clipboard", "capture", "order"],
            title: "采集顺序",
            value: (settings) => {
              return settings.clipboard.capture.order;
            },
          },
        ],
        title: "内容类型",
      },
      {
        description: "按复制来源决定哪些应用的内容可以进入历史。",
        id: "source",
        settings: [
          {
            control: { type: "appExclusion" },
            description: "从这些应用复制的内容不会进入历史。",
            id: "source.excludedApps",
            keywords: ["exclude", "ignore", "app", "source"],
            path: ["clipboard", "filters", "excludedAppIds"],
            title: "忽略应用",
            value: (settings) => {
              return settings.clipboard.filters.excludedAppIds;
            },
          },
        ],
        title: "应用过滤",
      },
      {
        description: "避免把密钥、Token 等敏感内容误存进历史。",
        id: "sensitive",
        settings: [
          {
            control: { type: "switch" },
            description: "检测到高置信度密钥、Token 等内容时跳过保存。",
            id: "sensitive.secretDetection",
            keywords: ["token", "key", "secret", "code"],
            path: ["clipboard", "sensitive", "secretDetection"],
            title: "识别密钥和 token",
            value: (settings) => {
              return settings.clipboard.sensitive.secretDetection;
            },
          },
        ],
        title: "隐私保护",
      },
    ],
    title: "采集",
  },
  {
    icon: "i-lucide:history",
    id: "organize",
    sections: [
      {
        description: "设置历史记录的保留时间和数量上限。",
        id: "history",
        settings: [
          {
            control: { type: "retention" },
            description:
              "超过该天数的普通记录会自动清理；收藏和置顶始终保留，0 表示不按时间清理。",
            id: "history.retention",
            keywords: ["retention", "cleanup", "history"],
            path: ["clipboard", "history", "retention"],
            title: "保留周期",
            value: (settings) => {
              return settings.clipboard.history.retention;
            },
          },
          {
            control: { min: 0, suffixKey: "items", type: "number" },
            description:
              "超过上限后，自动清理较旧的普通记录；收藏和置顶始终保留，0 表示不限数量。",
            id: "history.maxCount",
            keywords: ["max", "count", "limit"],
            path: ["clipboard", "history", "maxCount"],
            title: "最大保留条数",
            value: (settings) => {
              return settings.clipboard.history.maxCount;
            },
          },
          {
            control: { min: 0, suffixKey: "hours", type: "number" },
            description:
              "每隔指定小时检查并清理一次历史记录，0 表示关闭周期性清理；应用启动时仍会清理一次。",
            id: "history.cleanupIntervalHours",
            keywords: ["cleanup", "interval", "schedule"],
            path: ["clipboard", "history", "cleanupIntervalHours"],
            title: "自动清理周期",
            value: (settings) => {
              return settings.clipboard.history.cleanupIntervalHours;
            },
          },
        ],
        title: "保留与清理",
      },
      {
        description: "设置收藏、备注和分组相关行为。",
        id: "organizing",
        settings: [
          {
            control: { type: "switch" },
            description: "保存非空备注时，自动把该记录加入收藏。",
            id: "organizing.autoFavorite",
            keywords: ["note", "favorite", "auto"],
            path: ["clipboard", "content", "autoFavorite"],
            title: "有备注时自动收藏",
            value: (settings) => {
              return settings.clipboard.content.autoFavorite;
            },
          },
          {
            control: { label: "管理分组", type: "action" },
            description: "未来可按主题把历史记录归类到自定义分组。",
            disabled: true,
            id: "organizing.customGroups",
            keywords: ["group", "folder", "organize"],
            status: "comingSoon",
            title: "自定义分组",
          },
        ],
        title: "收藏与备注",
      },
      {
        description: "设置主窗口里的搜索习惯。",
        id: "search",
        settings: [
          {
            control: { type: "switch" },
            description: "打开剪贴板窗口时自动选中搜索框，方便直接输入关键词。",
            id: "search.defaultFocus",
            keywords: ["search", "focus", "open"],
            path: ["clipboard", "search", "defaultFocus"],
            title: "打开窗口时聚焦搜索",
            value: (settings) => {
              return settings.clipboard.search.defaultFocus;
            },
          },
          {
            control: { type: "switch" },
            description: "下次打开剪贴板窗口时清空搜索关键词，并显示完整列表。",
            id: "search.clearOnHide",
            keywords: ["search", "clear", "hide"],
            path: ["clipboard", "search", "clearOnHide"],
            title: "下次打开时清空搜索",
            value: (settings) => {
              return settings.clipboard.search.clearOnHide;
            },
          },
          {
            control: { options: CLIPBOARD_SORT_OPTIONS, type: "select" },
            description: "设置未搜索时历史列表的默认排序方式。",
            id: "search.sort",
            keywords: ["sort", "frequency", "usage", "created", "updated"],
            path: ["clipboard", "content", "sort"],
            title: "默认排序方式",
            value: (settings) => {
              return settings.clipboard.content.sort;
            },
          },
        ],
        title: "搜索与排序",
      },
    ],
    title: "历史",
  },
  {
    icon: "i-lucide:mouse-pointer-click",
    id: "reuse",
    sections: [
      {
        description: "设置点击历史记录时的复制、粘贴和格式处理方式。",
        id: "paste",
        settings: [
          {
            control: {
              options: CLICK_ACTION_OPTIONS,
              type: "segmented",
            },
            description: "选择左键点击历史记录时执行的动作。",
            id: "paste.autoPaste",
            keywords: ["paste", "click", "auto"],
            path: ["clipboard", "content", "autoPaste"],
            title: "左键点击",
            value: (settings) => {
              return settings.clipboard.content.autoPaste;
            },
          },
          {
            control: {
              options: MIDDLE_CLICK_ACTION_OPTIONS,
              type: "segmented",
            },
            description: "选择中键点击历史记录时执行的动作。",
            id: "paste.middleClick",
            keywords: ["paste", "middle click", "mouse"],
            path: ["clipboard", "content", "middleClick"],
            title: "中键点击",
            value: (settings) => {
              return settings.clipboard.content.middleClick;
            },
          },
          {
            control: { type: "switch" },
            description:
              "粘贴文本记录时默认去除 HTML/RTF 等格式，只粘贴纯文本。",
            id: "paste.plainDefault",
            keywords: ["plain", "paste", "format"],
            path: ["clipboard", "content", "pastePlain"],
            title: "默认纯文本粘贴",
            value: (settings) => {
              return settings.clipboard.content.pastePlain;
            },
          },
          {
            control: { type: "switch" },
            description: "粘贴文件记录时默认不粘贴文件本身，只粘贴文件路径。",
            id: "paste.fileMode",
            keywords: ["file", "path", "paste"],
            path: ["clipboard", "content", "pasteFilesAsPath"],
            title: "默认文件路径粘贴",
            value: (settings) => {
              return settings.clipboard.content.pasteFilesAsPath;
            },
          },
        ],
        title: "粘贴行为",
      },
      {
        description: "设置从历史复用内容时的复制格式和反馈。",
        id: "copy",
        settings: [
          {
            control: { type: "switch" },
            description:
              "复制文本记录时默认去除 HTML/RTF 等格式，只复制纯文本。",
            id: "copy.plainDefault",
            keywords: ["copy", "plain", "format"],
            path: ["clipboard", "content", "copyPlain"],
            title: "默认复制为纯文本",
            value: (settings) => {
              return settings.clipboard.content.copyPlain;
            },
          },
          {
            control: { type: "switch" },
            description: "从历史复制或粘贴记录时，刷新使用时间和使用次数。",
            id: "copy.updateOnReuse",
            keywords: ["copy", "paste", "reuse", "sort", "frequency"],
            path: ["clipboard", "content", "updateOnReuse"],
            title: "复用时更新记录",
            value: (settings) => {
              return settings.clipboard.content.updateOnReuse;
            },
          },
          {
            control: { type: "switch" },
            description: "剪贴板新内容成功进入历史时播放提示音。",
            id: "copy.sound",
            keywords: ["sound", "feedback", "copy"],
            path: ["clipboard", "feedback", "copySound"],
            title: "复制成功提示音",
            value: (settings) => {
              return settings.clipboard.feedback.copySound;
            },
          },
        ],
        title: "复制反馈",
      },
      {
        description: "控制历史记录行上显示哪些快捷操作。",
        id: "actions",
        settings: [
          {
            control: {
              label: "管理",
              options: ITEM_ACTION_OPTIONS,
              orderPath: ["clipboard", "content", "itemActionOrder"],
              type: "sortableCheckboxTree",
            },
            description: "选择鼠标悬停在记录上时显示的操作按钮。",
            id: "actions.visible",
            keywords: ["action", "hover", "buttons"],
            path: ["clipboard", "content", "itemActions"],
            title: "显示的快捷动作",
            value: (settings) => {
              return {
                order: settings.clipboard.content.itemActionOrder,
                selected: settings.clipboard.content.itemActions,
              };
            },
          },
          {
            control: { type: "switch" },
            description: "删除历史记录前先弹出确认，避免误删。",
            id: "actions.deleteConfirm",
            keywords: ["delete", "confirm"],
            path: ["clipboard", "content", "deleteConfirm"],
            title: "删除前确认",
            value: (settings) => {
              return settings.clipboard.content.deleteConfirm;
            },
          },
        ],
        title: "快捷动作",
      },
    ],
    title: "操作",
  },
  {
    icon: "i-lucide:panel-top",
    id: "workflow",
    sections: [
      {
        description: "设置剪贴板主窗口每次打开时的位置。",
        id: "window",
        settings: [
          {
            control: {
              options: [
                { label: "跟随光标", value: "followCursor" },
                { label: "屏幕中间", value: "center" },
                { label: "记住位置", value: "remember" },
              ],
              type: "segmented",
            },
            description: "选择每次打开剪贴板主窗口时的位置。",
            id: "window.position",
            keywords: ["window", "position", "cursor"],
            path: ["clipboard", "window", "position"],
            title: "打开位置",
            value: (settings) => {
              return settings.clipboard.window.position;
            },
          },
        ],
        title: "主窗口",
      },
      {
        description: "设置如何预览历史记录的完整内容。",
        id: "preview",
        settings: [
          {
            control: { type: "switch" },
            description: "鼠标悬停在记录上时显示完整内容预览。",
            id: "preview.hover",
            keywords: ["preview", "hover"],
            path: ["clipboard", "preview", "hoverEnabled"],
            title: "悬停预览",
            value: (settings) => {
              return settings.clipboard.preview.hoverEnabled;
            },
          },
          {
            control: {
              options: [
                { label: "300ms", value: "ms300" },
                { label: "500ms", value: "ms500" },
                { label: "1000ms", value: "ms1000" },
              ],
              type: "segmented",
            },
            description: "设置鼠标悬停多久后显示预览。",
            disabledWhen: (settings) => {
              return !settings.clipboard.preview.hoverEnabled;
            },
            id: "preview.delay",
            keywords: ["preview", "delay", "hover"],
            parentId: "preview.hover",
            path: ["clipboard", "preview", "hoverDelayMs"],
            title: "悬停延迟",
            value: (settings) => {
              return settings.clipboard.preview.hoverDelayMs;
            },
          },
          {
            control: { type: "switch" },
            description: "按住空格键时预览当前选中的记录。",
            id: "preview.space",
            keywords: ["space", "preview", "keyboard"],
            path: ["clipboard", "preview", "spaceEnabled"],
            title: "空格键预览",
            value: (settings) => {
              return settings.clipboard.preview.spaceEnabled;
            },
          },
        ],
        title: "内容预览",
      },
      {
        description: "设置界面主题、语言和列表显示密度。",
        id: "appearance",
        settings: [
          {
            control: {
              options: [
                { label: "跟随系统", value: "auto" },
                { label: "浅色", value: "light" },
                { label: "深色", value: "dark" },
              ],
              type: "segmented",
            },
            description: "选择界面使用浅色、深色，或跟随系统外观。",
            id: "appearance.theme",
            keywords: ["theme", "dark", "light"],
            path: ["appearance", "theme"],
            title: "主题",
            value: (settings) => {
              return settings.appearance.theme;
            },
          },
          {
            control: {
              options: [
                { label: "简体中文", value: "zh-CN" },
                { label: "English", value: "en-US" },
              ],
              type: "segmented",
            },
            description: "切换 EcoPaste 的界面语言。",
            id: "appearance.language",
            keywords: ["language", "locale", "english"],
            path: ["appearance", "language"],
            title: "语言",
            value: (settings) => {
              return settings.appearance.language;
            },
          },
          {
            control: { max: 5, min: 1, suffixKey: "lines", type: "number" },
            description: "限制文本记录在列表卡片中最多显示的行数。",
            id: "appearance.textMaxLines",
            keywords: ["density", "text", "line", "compact"],
            path: ["clipboard", "display", "textMaxLines"],
            title: "文本内容最大显示行数",
            value: (settings) => {
              return settings.clipboard.display.textMaxLines;
            },
          },
          {
            control: { max: 100, min: 20, suffixKey: "px", type: "number" },
            description: "限制图片记录在列表卡片中的缩略图高度。",
            id: "appearance.imageMaxHeight",
            keywords: ["density", "image", "height", "thumbnail"],
            path: ["clipboard", "display", "imageMaxHeight"],
            title: "图片最大显示高度",
            value: (settings) => {
              return settings.clipboard.display.imageMaxHeight;
            },
          },
          {
            control: { max: 5, min: 1, suffixKey: "files", type: "number" },
            description: "限制文件记录在列表卡片中最多显示的文件数量。",
            id: "appearance.fileMaxCount",
            keywords: ["density", "file", "count", "array"],
            path: ["clipboard", "display", "fileMaxCount"],
            title: "文件最多显示数量",
            value: (settings) => {
              return settings.clipboard.display.fileMaxCount;
            },
          },
          {
            control: { label: "跟随系统", type: "status" },
            description: "跟随系统的减少动态效果设置，降低动画强度。",
            id: "appearance.reducedMotion",
            keywords: ["motion", "accessibility"],
            status: "alwaysOn",
            title: "减少动画",
          },
        ],
        title: "外观与语言",
      },
      {
        description: "控制 EcoPaste 是否随系统启动，以及显示哪些系统入口。",
        id: "control",
        settings: [
          {
            control: { type: "switch" },
            description: "登录后自动启动 EcoPaste。",
            id: "control.autoStart",
            keywords: ["startup", "login", "autostart"],
            path: ["general", "autoStart"],
            title: "登录时启动",
            value: (settings) => {
              return settings.general.autoStart;
            },
          },
          {
            control: { type: "switch" },
            description: "开机启动时只在后台运行，不自动弹出剪贴板窗口。",
            disabledWhen: (settings) => {
              return !settings.general.autoStart;
            },
            id: "control.silentStart",
            keywords: ["startup", "silent", "login"],
            parentId: "control.autoStart",
            path: ["general", "silentStart"],
            title: "静默启动",
            value: (settings) => {
              return settings.general.silentStart;
            },
          },
          {
            control: { type: "switch" },
            description: "在菜单栏显示 EcoPaste 图标。",
            id: "control.trayIcon",
            keywords: ["tray", "menu bar", "system"],
            path: ["general", "trayIcon"],
            title: "菜单栏图标",
            value: (settings) => {
              return settings.general.trayIcon;
            },
          },
          {
            control: { type: "switch" },
            description: "在程序坞显示 EcoPaste 图标。",
            id: "control.dockIcon",
            keywords: ["dock", "taskbar", "icon"],
            path: ["general", "dockIcon"],
            title: "程序坞图标",
            value: (settings) => {
              return settings.general.dockIcon;
            },
          },
        ],
        title: "系统入口",
      },
    ],
    title: "界面",
  },
  {
    icon: "i-lucide:keyboard",
    id: "shortcuts",
    sections: [
      {
        description: "设置在任意应用中都能触发的系统快捷键。",
        id: "globalShortcuts",
        settings: [
          {
            control: { placeholder: "Alt+C", type: "text" },
            description: "在任意应用中打开或隐藏剪贴板主窗口。",
            id: "shortcuts.openClipboard",
            keywords: ["shortcut", "hotkey", "open"],
            path: ["shortcuts", "openClipboard"],
            title: "打开主窗口",
            value: (settings) => {
              return settings.shortcuts.openClipboard;
            },
          },
          {
            control: { placeholder: "Alt+X", type: "text" },
            description: "在任意应用中打开或隐藏偏好设置窗口。",
            id: "shortcuts.openPreference",
            keywords: ["shortcut", "hotkey", "preference"],
            path: ["shortcuts", "openPreference"],
            title: "打开偏好设置",
            value: (settings) => {
              return settings.shortcuts.openPreference;
            },
          },
        ],
        title: "全局快捷键",
      },
    ],
    title: "快捷键",
  },
  {
    icon: "i-lucide:database",
    id: "data",
    sections: [
      {
        description: "打开 EcoPaste 在本机保存数据、资源和日志的位置。",
        id: "localData",
        settings: [
          {
            control: { label: "打开", type: "action" },
            description: "打开历史数据库、设置文件和资源缓存所在目录。",
            id: "localData.dataDirectory",
            keywords: ["database", "sqlite", "local", "cache", "image", "icon"],
            title: "数据目录",
          },
          {
            control: { label: "打开", type: "action" },
            description: "打开用于排查问题的运行日志目录。",
            id: "localData.logDirectory",
            keywords: ["log", "diagnostic"],
            title: "日志目录",
          },
          {
            control: { label: "清理缓存", type: "action" },
            description: "移除不再被历史记录引用的图片、缩略图和图标缓存。",
            id: "localData.cleanCache",
            keywords: ["cache", "clean", "storage"],
            title: "清理缓存",
          },
        ],
        title: "存储位置",
      },
      {
        description: "导出或导入 EcoPaste 备份包。",
        id: "backup",
        settings: [
          {
            control: { label: "导出", type: "action" },
            description: "导出 .ecopastebak 备份包，默认使用密码加密。",
            id: "backup.exportHistory",
            keywords: ["export", "backup", "history"],
            title: "导出备份",
          },
          {
            control: { label: "导入", type: "action" },
            description: "导入 .ecopastebak 备份包，支持合并或覆盖恢复。",
            id: "backup.importHistory",
            keywords: ["import", "backup", "history"],
            title: "导入备份",
          },
        ],
        title: "备份迁移",
      },
      {
        description: "恢复偏好设置默认值。",
        id: "diagnostics",
        settings: [
          {
            control: { danger: true, label: "重置", type: "action" },
            description: "恢复所有偏好默认值，同时保留历史记录。",
            id: "diagnostics.resetPreferences",
            keywords: ["reset", "preferences"],
            title: "重置所有偏好",
          },
        ],
        title: "诊断恢复",
      },
      {
        description: "未来可授权外部工具只读访问本地历史。",
        id: "external",
        settings: [
          {
            control: { label: "配置 CLI", type: "action" },
            description: "未来可通过命令行查询本地剪贴板历史。",
            disabled: true,
            id: "external.cli",
            keywords: ["cli", "terminal", "json"],
            status: "comingSoon",
            title: "CLI 访问",
          },
          {
            control: { label: "配置 AI 访问", type: "action" },
            description: "未来可允许 AI 工具搜索本地剪贴板历史。",
            disabled: true,
            id: "external.ai",
            keywords: ["ai", "agent", "tool"],
            status: "comingSoon",
            title: "AI 工具访问",
          },
          {
            control: { label: "配置 MCP", type: "action" },
            description: "未来可为 MCP 客户端提供只读历史查询工具。",
            disabled: true,
            id: "external.mcp",
            keywords: ["mcp", "server", "agent"],
            status: "comingSoon",
            title: "MCP Server",
          },
        ],
        title: "外部工具",
      },
      {
        description: "设置 EcoPaste 如何检查和接收新版本。",
        id: "updates",
        settings: [
          {
            control: { type: "switch" },
            description: "定期检查 EcoPaste 是否有可用的新版本。",
            id: "updates.autoCheck",
            keywords: ["update", "version"],
            path: ["update", "autoCheck"],
            title: "自动检查更新",
            value: (settings) => {
              return settings.update.autoCheck;
            },
          },
          {
            control: { type: "switch" },
            description: "检查更新时同时接收 Beta 测试版本。",
            disabledWhen: (settings) => {
              return !settings.update.autoCheck;
            },
            id: "updates.beta",
            keywords: ["beta", "update"],
            parentId: "updates.autoCheck",
            path: ["update", "includeBeta"],
            title: "包含 Beta 版本",
            value: (settings) => {
              return settings.update.includeBeta;
            },
          },
          {
            control: {
              options: [
                { label: "每天", value: "daily" },
                { label: "每周", value: "weekly" },
                { label: "每月", value: "monthly" },
              ],
              type: "segmented",
            },
            description: "设置自动检查更新的时间间隔。",
            disabledWhen: (settings) => {
              return !settings.update.autoCheck;
            },
            id: "updates.frequency",
            keywords: ["update", "frequency", "schedule"],
            parentId: "updates.autoCheck",
            path: ["update", "frequency"],
            title: "自动检查频率",
            value: (settings) => {
              return settings.update.frequency;
            },
          },
        ],
        title: "更新",
      },
    ],
    title: "数据",
  },
  {
    icon: "i-lucide:info",
    id: "about",
    sections: [],
    title: "关于",
  },
];

export const allPreferenceSettings = preferenceTabs.flatMap((tab) => {
  return tab.sections.flatMap((section) => {
    return section.settings.map((setting) => {
      return { section, setting, tab };
    });
  });
});
