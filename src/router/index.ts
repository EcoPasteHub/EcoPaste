// @unocss-include
import { createBrowserRouter } from "react-router-dom";

import DefaultLayout from "@/layouts/Default";
import About from "@/pages/About";
import ClipboardHistory from "@/pages/Clipboard/History";
import ClipboardSettings from "@/pages/Clipboard/Settings";
import DataBackup from "@/pages/DataBackup";
import Settings from "@/pages/Settings";
import type { Route } from "@/types/router";

export const routes: Route[] = [
	{
		path: "/",
		Component: DefaultLayout,
		children: [
			{
				path: "/",
				Component: ClipboardSettings,
				meta: {
					title: "剪切板",
					icon: "i-lucide:clipboard-list",
				},
			},
			{
				path: "/settings",
				Component: Settings,
				meta: {
					title: "通用设置",
					icon: "i-lucide:bolt",
				},
			},
			{
				path: "/data-backup",
				Component: DataBackup,
				meta: {
					title: "数据备份",
					icon: "i-lucide:database-backup",
				},
			},
			{
				path: "/about",
				Component: About,
				meta: {
					title: "关于",
					icon: "i-lucide:info",
				},
			},
		],
	},
	{
		path: "/clipboard-history",
		Component: ClipboardHistory,
		meta: {
			windowOptions: {
				width: 360,
				height: 600,
				resizable: false,
				maximizable: false,
				decorations: false,
				visible: false,
				transparent: true,
				alwaysOnTop: true,
				acceptFirstMouse: true,
			},
		},
	},
];

export const router = createBrowserRouter(routes);
