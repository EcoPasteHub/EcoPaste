// @unocss-include
import { createBrowserRouter } from "react-router-dom";

import Preference from "@/layouts/Preference";
import About from "@/pages/About";
import Backup from "@/pages/Backup";
import ClipboardHistory from "@/pages/Clipboard/History";
import ClipboardSettings from "@/pages/Clipboard/Settings";
import General from "@/pages/General";
import History from "@/pages/History";
import Shortcut from "@/pages/Shortcut";
import type { Route } from "@/types/router";

export const preferenceRoute: Route = {
	path: "/preference",
	Component: Preference,
	children: [
		{
			path: "clipboard",
			Component: ClipboardSettings,
			meta: {
				title: "preference.menu.title.clipboard",
				icon: "i-lucide:clipboard-list",
			},
		},
		{
			path: "history",
			Component: History,
			meta: {
				title: "preference.menu.title.history",
				icon: "i-lucide:history",
			},
		},
		{
			path: "general",
			Component: General,
			meta: {
				title: "preference.menu.title.general",
				icon: "i-lucide:bolt",
			},
		},
		{
			path: "shortcut",
			Component: Shortcut,
			meta: {
				title: "preference.menu.title.shortcut",
				icon: "i-lucide:keyboard",
			},
		},
		{
			path: "backup",
			Component: Backup,
			meta: {
				title: "preference.menu.title.backup",
				icon: "i-lucide:database-backup",
			},
		},
		{
			path: "about",
			Component: About,
			meta: {
				title: "preference.menu.title.about",
				icon: "i-lucide:info",
			},
		},
	],
	meta: {
		windowOptions: {
			width: 700,
			height: 480,
			center: true,
			resizable: false,
			maximizable: false,
			hiddenTitle: true,
			visible: false,
			transparent: true,
			titleBarStyle: "overlay",
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
