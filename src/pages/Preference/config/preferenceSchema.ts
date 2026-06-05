/* @unocss-include */
import { isMac } from "@/utils/is";
import type { PreferenceTab } from "../types/preferences";

const MOD_KEY = isMac ? "⌘" : "Ctrl";
const ENTER_KEY = isMac ? "⏎" : "Enter";
const BACKSPACE_KEY = "⌫";
const UP_KEY = "↑";
const DOWN_KEY = "↓";
const CLICK_ACTION_OPTIONS = [
  { label: "禁用", value: "disabled" },
  { label: "单击粘贴", value: "singleClickPaste" },
  { label: "双击粘贴", value: "doubleClickPaste" },
  { label: "单击复制", value: "singleClickCopy" },
  { label: "双击复制", value: "doubleClickCopy" },
];

export const preferenceTabs: PreferenceTab[] = [
  {
    icon: "i-lucide:clipboard-plus",
    id: "record",
    sections: [
      {
        description: "决定哪些剪贴板类型会进入历史。",
        id: "capture",
        settings: [
          {
            control: { type: "switch" },
            description: "关闭后不再保存纯文本内容。",
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
            description: "关闭后富文本网页内容只按可用纯文本处理。",
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
            description: "关闭后文档富文本只按可用纯文本处理。",
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
            description: "关闭后不再保存剪贴板图片。",
            id: "capture.images",
            keywords: ["image", "picture", "thumbnail"],
            path: ["clipboard", "capture", "images"],
            title: "图片",
            value: (settings) => {
              return settings.clipboard.capture.images;
            },
          },
          {
            control: { type: "switch" },
            description: "关闭后不再保存复制的文件和文件夹。",
            id: "capture.files",
            keywords: ["file", "folder", "path"],
            path: ["clipboard", "capture", "files"],
            title: "文件和文件夹",
            value: (settings) => {
              return settings.clipboard.capture.files;
            },
          },
        ],
        title: "内容类型",
      },
      {
        description: "按来源应用控制记录范围。",
        id: "source",
        settings: [
          {
            control: { type: "appExclusion" },
            description: "这些应用中的复制不会保存。",
            id: "source.excludedApps",
            keywords: ["exclude", "ignore", "app", "source"],
            path: ["clipboard", "filters", "excludedAppIds"],
            title: "忽略应用",
            value: (settings) => {
              return settings.clipboard.filters.excludedAppIds;
            },
          },
          {
            control: { label: "已启用", type: "status" },
            description: "在记录中显示复制来源。",
            id: "source.detectApp",
            keywords: ["source", "app", "origin"],
            status: "alwaysOn",
            title: "识别来源应用",
          },
          {
            control: { placeholder: "每行一个目录", type: "textarea" },
            description: "补充应用搜索范围，用于识别名称和图标。",
            id: "source.scanDirs",
            keywords: ["scan", "applications", "directory"],
            path: ["clipboard", "filters", "scanDirs"],
            title: "应用发现目录",
            value: (settings) => {
              return settings.clipboard.filters.scanDirs;
            },
          },
          {
            control: { label: "配置规则", type: "action" },
            description: "为不同应用单独设置保存策略。",
            disabled: true,
            id: "source.perAppRules",
            keywords: ["per app", "rule", "source"],
            status: "comingSoon",
            title: "单应用规则",
          },
        ],
        title: "来源应用",
      },
      {
        description: "减少误存隐私内容的风险。",
        id: "sensitive",
        settings: [
          {
            control: { label: "默认保护", type: "status" },
            description: "密码管理器默认不记录。",
            id: "sensitive.passwordApps",
            keywords: ["password", "private", "secret"],
            status: "alwaysOn",
            title: "默认忽略密码应用",
          },
          {
            control: { type: "switch" },
            description: "发现密钥、验证码等内容时不保存。",
            disabled: true,
            id: "sensitive.secretDetection",
            keywords: ["token", "key", "secret", "code"],
            status: "comingSoon",
            title: "识别密钥和 token",
          },
          {
            control: { label: "管理规则", type: "action" },
            description: "按关键词或模式跳过内容。",
            disabled: true,
            id: "sensitive.keywordRules",
            keywords: ["keyword", "pattern", "ignore"],
            status: "comingSoon",
            title: "内容忽略规则",
          },
          {
            control: { label: "进入隐私模式", type: "action" },
            description: "短时间关闭剪贴板记录。",
            disabled: true,
            id: "sensitive.privateMode",
            keywords: ["private", "pause", "temporary"],
            status: "comingSoon",
            title: "临时暂停记录",
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
        description: "限制历史规模，避免长期堆积。",
        id: "history",
        settings: [
          {
            control: { type: "retention" },
            description: "超过时间的普通记录会自动清理。",
            id: "history.retention",
            keywords: ["retention", "cleanup", "history"],
            path: ["clipboard", "history", "retention"],
            title: "保留周期",
            value: (settings) => {
              return settings.clipboard.history.retention;
            },
          },
          {
            control: { min: 0, type: "number" },
            description: "超过数量后，仅保留最近记录。",
            id: "history.maxCount",
            keywords: ["max", "count", "limit"],
            path: ["clipboard", "history", "maxCount"],
            title: "最大保留条数",
            value: (settings) => {
              return settings.clipboard.history.maxCount;
            },
          },
          {
            control: { min: 0, suffix: "GB", type: "number" },
            description: "跳过过大的剪贴板内容。",
            disabled: true,
            id: "history.maxSize",
            keywords: ["size", "limit", "large"],
            status: "comingSoon",
            title: "最大记录大小",
          },
          {
            control: { label: "始终保留", type: "status" },
            description: "收藏记录不会被自动清理。",
            id: "history.keepFavorite",
            keywords: ["favorite", "cleanup", "keep"],
            status: "alwaysOn",
            title: "清理时保留收藏",
          },
          {
            control: { label: "始终保留", type: "status" },
            description: "置顶记录不会被自动清理。",
            id: "history.keepPinned",
            keywords: ["pinned", "cleanup", "keep"],
            status: "alwaysOn",
            title: "清理时保留置顶",
          },
        ],
        title: "保留与清理",
      },
      {
        description: "标记需要长期保留的记录。",
        id: "organizing",
        settings: [
          {
            control: { label: "已支持", type: "status" },
            description: "把常用内容放进收藏视图。",
            id: "organizing.favorite",
            keywords: ["favorite", "star"],
            status: "alwaysOn",
            title: "收藏记录",
          },
          {
            control: { label: "待开放入口", type: "status" },
            description: "让重要记录保持在列表前面。",
            id: "organizing.pinned",
            keywords: ["pin", "pinned"],
            status: "requiresBackend",
            title: "置顶记录",
          },
          {
            control: { label: "已支持", type: "status" },
            description: "给记录补充上下文。",
            id: "organizing.notes",
            keywords: ["note", "memo", "comment"],
            status: "alwaysOn",
            title: "记录备注",
          },
          {
            control: { type: "switch" },
            description: "有备注的记录自动加入收藏。",
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
            description: "按主题归类保存的记录。",
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
        description: "设置主窗口里的查找习惯。",
        id: "search",
        settings: [
          {
            control: { type: "switch" },
            description: "打开主窗口后直接输入关键词。",
            id: "search.defaultFocus",
            keywords: ["search", "focus", "open"],
            path: ["clipboard", "search", "defaultFocus"],
            title: "打开时聚焦搜索",
            value: (settings) => {
              return settings.clipboard.search.defaultFocus;
            },
          },
          {
            control: { type: "switch" },
            description: "下次打开时回到完整列表。",
            id: "search.clearOnHide",
            keywords: ["search", "clear", "hide"],
            path: ["clipboard", "search", "clearOnHide"],
            title: "关闭后清空搜索",
            value: (settings) => {
              return settings.clipboard.search.clearOnHide;
            },
          },
          {
            control: { type: "switch" },
            description: "常用记录优先显示。",
            id: "search.frequencySort",
            keywords: ["sort", "frequency", "usage"],
            path: ["clipboard", "content", "autoSortByFrequency"],
            title: "按使用频率排序",
            value: (settings) => {
              return settings.clipboard.content.autoSortByFrequency;
            },
          },
        ],
        title: "查找与排序",
      },
    ],
    title: "历史",
  },
  {
    icon: "i-lucide:mouse-pointer-click",
    id: "reuse",
    sections: [
      {
        description: "调整鼠标和粘贴格式的默认行为。",
        id: "paste",
        settings: [
          {
            control: {
              options: CLICK_ACTION_OPTIONS,
              type: "segmented",
            },
            description: "设置左键点击记录时的动作。",
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
              options: CLICK_ACTION_OPTIONS,
              type: "segmented",
            },
            description: "设置右键点击记录时的动作。",
            disabled: true,
            id: "paste.secondaryClick",
            keywords: ["paste", "right click", "context menu"],
            status: "comingSoon",
            title: "右键点击",
            value: () => {
              return "disabled";
            },
          },
          {
            control: {
              options: CLICK_ACTION_OPTIONS,
              type: "segmented",
            },
            description: "设置中键点击记录时的动作。",
            disabled: true,
            id: "paste.middleClick",
            keywords: ["paste", "middle click", "mouse"],
            status: "comingSoon",
            title: "中键点击",
            value: () => {
              return "disabled";
            },
          },
          {
            control: { type: "switch" },
            description: "默认只粘贴文字内容。",
            id: "paste.plainDefault",
            keywords: ["plain", "paste", "format"],
            path: ["clipboard", "content", "pastePlain"],
            title: "默认纯文本粘贴",
            value: (settings) => {
              return settings.clipboard.content.pastePlain;
            },
          },
          {
            control: {
              options: [
                { label: "文件", value: "files" },
                { label: "路径", value: "paths" },
              ],
              type: "segmented",
            },
            description: "选择粘贴文件本身或路径文本。",
            disabled: true,
            id: "paste.fileMode",
            keywords: ["file", "path", "paste"],
            status: "comingSoon",
            title: "文件粘贴方式",
          },
        ],
        title: "粘贴行为",
      },
      {
        description: "设置复制格式和提示反馈。",
        id: "copy",
        settings: [
          {
            control: { type: "switch" },
            description: "默认只复制文字内容。",
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
            description: "记录新内容时播放提示音。",
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
        description: "控制记录旁边出现哪些操作。",
        id: "actions",
        settings: [
          {
            control: {
              mode: "multiple",
              options: [
                { label: "复制", value: "copy" },
                { label: "纯文本粘贴", value: "pastePlain" },
                { label: "备注", value: "note" },
                { label: "收藏", value: "star" },
                { label: "删除", value: "delete" },
              ],
              type: "select",
            },
            description: "选择悬停时显示的按钮。",
            id: "actions.visible",
            keywords: ["action", "hover", "buttons"],
            path: ["clipboard", "content", "itemActions"],
            title: "显示的快捷动作",
            value: (settings) => {
              return settings.clipboard.content.itemActions;
            },
          },
          {
            control: { type: "switch" },
            description: "删除前先确认一次。",
            id: "actions.deleteConfirm",
            keywords: ["delete", "confirm"],
            path: ["clipboard", "content", "deleteConfirm"],
            title: "删除前确认",
            value: (settings) => {
              return settings.clipboard.content.deleteConfirm;
            },
          },
          {
            control: { label: "按内容自动出现", type: "status" },
            description: "按内容显示打开、发送或定位等动作。",
            id: "actions.contextual",
            keywords: ["open", "link", "email", "reveal"],
            status: "alwaysOn",
            title: "智能动作",
          },
          {
            control: { label: "管理转换", type: "action" },
            description: "添加自定义处理动作。",
            disabled: true,
            id: "actions.transforms",
            keywords: ["transform", "custom", "automation"],
            status: "comingSoon",
            title: "自定义转换",
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
        description: "设置主窗口出现的位置。",
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
            description: "选择每次打开主窗口的位置。",
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
        description: "快速查看记录的完整内容。",
        id: "preview",
        settings: [
          {
            control: { type: "switch" },
            description: "鼠标停留时显示预览。",
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
            description: "设置预览出现前的等待时间。",
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
            description: "按住空格查看选中记录。",
            id: "preview.space",
            keywords: ["space", "preview", "keyboard"],
            path: ["clipboard", "preview", "spaceEnabled"],
            title: "空格键预览",
            value: (settings) => {
              return settings.clipboard.preview.spaceEnabled;
            },
          },
          {
            control: { type: "switch" },
            description: "查看富文本的原始内容。",
            id: "preview.original",
            keywords: ["original", "html", "source"],
            path: ["clipboard", "content", "showOriginalPreview"],
            title: "显示原始内容",
            value: (settings) => {
              return settings.clipboard.content.showOriginalPreview;
            },
          },
        ],
        title: "内容预览",
      },
      {
        description: "设置界面的视觉偏好。",
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
            description: "选择浅色、深色或跟随系统。",
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
            description: "切换界面语言。",
            id: "appearance.language",
            keywords: ["language", "locale", "english"],
            path: ["appearance", "language"],
            title: "语言",
            value: (settings) => {
              return settings.appearance.language;
            },
          },
          {
            control: {
              options: [
                { label: "舒适", value: "comfortable" },
                { label: "紧凑", value: "compact" },
              ],
              type: "segmented",
            },
            description: "切换舒适或紧凑布局。",
            disabled: true,
            id: "appearance.density",
            keywords: ["density", "compact"],
            status: "comingSoon",
            title: "信息密度",
          },
          {
            control: { label: "跟随系统", type: "status" },
            description: "跟随系统减少动态效果。",
            id: "appearance.reducedMotion",
            keywords: ["motion", "accessibility"],
            status: "alwaysOn",
            title: "减少动画",
          },
        ],
        title: "外观与语言",
      },
      {
        description: "控制 EcoPaste 在系统中的入口。",
        id: "control",
        settings: [
          {
            control: { type: "switch" },
            description: "登录系统后自动运行。",
            id: "control.autoStart",
            keywords: ["startup", "login", "autostart"],
            path: ["general", "autoStart"],
            title: "开机启动",
            value: (settings) => {
              return settings.general.autoStart;
            },
          },
          {
            control: { type: "switch" },
            description: "后台启动，不弹出主窗口。",
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
            description: "显示常驻系统入口。",
            id: "control.trayIcon",
            keywords: ["tray", "menu bar", "system"],
            path: ["general", "trayIcon"],
            title: "菜单栏 / 托盘图标",
            value: (settings) => {
              return settings.general.trayIcon;
            },
          },
          {
            control: { type: "switch" },
            description: "显示桌面任务入口。",
            id: "control.dockIcon",
            keywords: ["dock", "taskbar", "icon"],
            path: ["general", "dockIcon"],
            title: "Dock / 任务栏图标",
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
        description: "可在任意应用中使用。",
        id: "globalShortcuts",
        settings: [
          {
            control: { placeholder: "Alt+C", type: "text" },
            description: "打开或隐藏剪贴板主窗口。",
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
            description: "打开或隐藏偏好设置。",
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
      {
        description: "这些快捷键仅展示，不能修改。",
        id: "windowShortcuts",
        settings: [
          {
            control: {
              shortcuts: [{ keys: [ENTER_KEY], label: "粘贴选中项" }],
              type: "shortcutTags",
            },
            description: "粘贴当前选中的记录。",
            id: "shortcuts.windowPaste",
            keywords: ["shortcut", "paste", "enter"],
            title: "粘贴选中项",
          },
          {
            control: {
              shortcuts: [
                {
                  keys: [MOD_KEY, ENTER_KEY],
                  label: "粘贴选中项为纯文本",
                },
              ],
              type: "shortcutTags",
            },
            description: "粘贴当前记录的纯文本版本。",
            id: "shortcuts.windowPastePlain",
            keywords: ["shortcut", "paste", "plain", "enter"],
            title: "粘贴选中项为纯文本",
          },
          {
            control: {
              shortcuts: [
                {
                  keys: [MOD_KEY, BACKSPACE_KEY],
                  label: "删除选中项",
                },
              ],
              type: "shortcutTags",
            },
            description: "删除当前选中的记录。",
            id: "shortcuts.windowDelete",
            keywords: ["shortcut", "delete", "backspace"],
            title: "删除选中项",
          },
          {
            control: {
              shortcuts: [{ keys: [MOD_KEY, "D"], label: "收藏 / 取消收藏" }],
              type: "shortcutTags",
            },
            description: "切换当前记录的收藏状态。",
            id: "shortcuts.windowFavorite",
            keywords: ["shortcut", "favorite", "star"],
            title: "收藏 / 取消收藏",
          },
          {
            control: {
              shortcuts: [{ keys: [UP_KEY, "/", DOWN_KEY], label: "上下导航" }],
              type: "shortcutTags",
            },
            description: "移动列表选中项。",
            id: "shortcuts.windowNavigate",
            keywords: ["shortcut", "navigate", "arrow"],
            title: "上下导航",
          },
          {
            control: {
              shortcuts: [
                {
                  keys: [MOD_KEY, "N"],
                  label: "粘贴第 N 项（N = 1…9, 0）",
                },
              ],
              type: "shortcutTags",
            },
            description: "按序号快速粘贴可见记录。",
            id: "shortcuts.windowQuickPaste",
            keywords: ["shortcut", "quick paste", "number"],
            title: "粘贴第 N 项（N = 1…9, 0）",
          },
          {
            control: {
              shortcuts: [{ keys: [MOD_KEY, "F"], label: "聚焦搜索框" }],
              type: "shortcutTags",
            },
            description: "开始搜索历史记录。",
            id: "shortcuts.windowFocusSearch",
            keywords: ["shortcut", "search", "focus"],
            title: "聚焦搜索框",
          },
          {
            control: {
              shortcuts: [
                { keys: [MOD_KEY, "P"], label: "固定 / 取消固定窗口" },
              ],
              type: "shortcutTags",
            },
            description: "让主窗口保持显示或恢复自动隐藏。",
            id: "shortcuts.windowPin",
            keywords: ["shortcut", "pin", "window"],
            title: "固定 / 取消固定窗口",
          },
          {
            control: {
              shortcuts: [{ keys: [MOD_KEY, ","], label: "打开偏好设置" }],
              type: "shortcutTags",
            },
            description: "打开偏好设置窗口。",
            id: "shortcuts.windowOpenPreference",
            keywords: ["shortcut", "preference", "settings"],
            title: "打开偏好设置",
          },
        ],
        title: "主窗口快捷键",
      },
    ],
    title: "快捷键",
  },
  {
    icon: "i-lucide:database",
    id: "data",
    sections: [
      {
        description: "打开本机数据和日志目录。",
        id: "localData",
        settings: [
          {
            control: { label: "打开", type: "action" },
            description: "打开历史、设置和资源所在目录。",
            id: "localData.dataDirectory",
            keywords: ["database", "sqlite", "local", "cache", "image", "icon"],
            title: "数据目录",
          },
          {
            control: { label: "打开", type: "action" },
            description: "打开运行日志目录。",
            id: "localData.logDirectory",
            keywords: ["log", "diagnostic"],
            title: "日志目录",
          },
          {
            control: { label: "清理缓存", type: "action" },
            description: "移除未使用的资源缓存。",
            disabled: true,
            id: "localData.cleanCache",
            keywords: ["cache", "clean", "storage"],
            status: "comingSoon",
            title: "清理缓存",
          },
        ],
        title: "存储位置",
      },
      {
        description: "导入或导出历史数据。",
        id: "backup",
        settings: [
          {
            control: { label: "导出", type: "action" },
            description: "保存历史和资源备份。",
            disabled: true,
            id: "backup.exportHistory",
            keywords: ["export", "backup", "history"],
            status: "comingSoon",
            title: "导出历史",
          },
          {
            control: { label: "导入", type: "action" },
            description: "从备份恢复历史。",
            disabled: true,
            id: "backup.importHistory",
            keywords: ["import", "backup", "history"],
            status: "comingSoon",
            title: "导入历史",
          },
        ],
        title: "备份迁移",
      },
      {
        description: "恢复窗口和偏好默认状态。",
        id: "diagnostics",
        settings: [
          {
            control: { label: "重置", type: "action" },
            description: "清除已保存的窗口位置。",
            disabled: true,
            id: "diagnostics.resetWindow",
            keywords: ["reset", "window"],
            status: "comingSoon",
            title: "重置窗口位置",
          },
          {
            control: { danger: true, label: "重置", type: "action" },
            description: "恢复默认设置，保留历史记录。",
            disabled: true,
            id: "diagnostics.resetPreferences",
            keywords: ["reset", "preferences"],
            status: "comingSoon",
            title: "重置所有偏好",
          },
        ],
        title: "诊断恢复",
      },
      {
        description: "授权外部工具读取本地历史。",
        id: "external",
        settings: [
          {
            control: { label: "配置 CLI", type: "action" },
            description: "通过命令行查询历史。",
            disabled: true,
            id: "external.cli",
            keywords: ["cli", "terminal", "json"],
            status: "comingSoon",
            title: "CLI 访问",
          },
          {
            control: { label: "配置 AI 访问", type: "action" },
            description: "允许 AI 工具搜索历史。",
            disabled: true,
            id: "external.ai",
            keywords: ["ai", "agent", "tool"],
            status: "comingSoon",
            title: "AI 工具访问",
          },
          {
            control: { label: "配置 MCP", type: "action" },
            description: "为 MCP 客户端提供只读工具。",
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
        description: "设置版本检查方式。",
        id: "updates",
        settings: [
          {
            control: { type: "switch" },
            description: "自动检查新版本。",
            id: "updates.autoCheck",
            keywords: ["update", "version"],
            path: ["update", "autoCheck"],
            title: "自动检查更新",
            value: (settings) => {
              return settings.update.autoCheck;
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
            description: "设置自动检查的间隔。",
            disabled: true,
            id: "updates.frequency",
            keywords: ["update", "frequency", "schedule"],
            status: "comingSoon",
            title: "自动检查频率",
            value: () => {
              return "weekly";
            },
          },
          {
            control: { type: "switch" },
            description: "同时接收测试版更新。",
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
