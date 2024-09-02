// @unocss-include
import Preference from "@/layouts/Preference";
import About from "@/pages/About";
import Backup from "@/pages/Backup";
import ClipboardPanel from "@/pages/Clipboard/Panel";
import ClipboardSettings from "@/pages/Clipboard/Settings";
import General from "@/pages/General";
import History from "@/pages/History";
import Shortcut from "@/pages/Shortcut";
import type { Route } from "@/types/router";
import { createHashRouter } from "react-router-dom";

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
};

export const routes: Route[] = [
	preferenceRoute,
	{
		path: "/",
		Component: ClipboardPanel,
	},
];

export const router = createHashRouter(routes);
