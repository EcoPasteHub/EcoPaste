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
				title: "preference.menu.title.clipboard",
				icon: "i-lucide:clipboard-list",
			},
		},
		{
			path: "settings",
			Component: Settings,
			meta: {
				title: "preference.menu.title.settings",
				icon: "i-lucide:bolt",
			},
		},
		{
			path: "data-backup",
			Component: DataBackup,
			meta: {
				title: "preference.menu.title.data_backup",
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
};

export const routes: Route[] = [
	preferenceRoute,
	{
		path: "/",
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
