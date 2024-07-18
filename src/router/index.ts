// @unocss-include
import { createBrowserRouter } from "react-router-dom";

import Preference from "@/layouts/Preference";
import About from "@/pages/About";
import ClipboardHistory from "@/pages/Clipboard/History";
import ClipboardSettings from "@/pages/Clipboard/Settings";
import DataBackup from "@/pages/DataBackup";
import Settings from "@/pages/Settings";
import type { Route } from "@/types/router";

export const preferenceRoute: Route = {
	path: "/preference",
	Component: Preference,
	children: [
		{
			path: "clipboard",
			Component: ClipboardSettings,
			meta: {
				title: "剪切板",
				icon: "i-lucide:clipboard-list",
			},
		},
		{
			path: "settings",
			Component: Settings,
			meta: {
				title: "通用设置",
				icon: "i-lucide:bolt",
			},
		},
		{
			path: "data-backup",
			Component: DataBackup,
			meta: {
				title: "数据备份",
				icon: "i-lucide:database-backup",
			},
		},
		{
			path: "about",
			Component: About,
			meta: {
				title: "关于",
				icon: "i-lucide:info",
			},
		},
	],
	meta: {
		windowOptions: {
			title: "偏好设置",
			center: true,
			resizable: false,
			maximizable: false,
			hiddenTitle: true,
			titleBarStyle: "Overlay",
		},
	},
};

export const routes: Route[] = [
	preferenceRoute,
	{
		path: "/",
		Component: ClipboardHistory,
	},
];

export const router = createBrowserRouter(routes);
