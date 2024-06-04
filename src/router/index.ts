// @unocss-include
import { createBrowserRouter } from "react-router-dom";

import DefaultLayout from "@/layouts/Default";
import About from "@/pages/About";
import Clipboard from "@/pages/Clipboard";
import Settings from "@/pages/Settings";
import type { Route } from "@/types/router";

export const routes: Route[] = [
	{
		path: "/",
		Component: DefaultLayout,
		children: [
			{
				path: "/",
				Component: Clipboard,
				meta: {
					title: "剪切板",
					icon: "i-lucide-clipboard-list",
				},
			},
			{
				path: "/settings",
				Component: Settings,
				meta: {
					title: "通用设置",
					icon: "i-lucide-settings",
				},
			},
			{
				path: "/about",
				Component: About,
				meta: {
					title: "关于",
					icon: "i-lucide-info",
				},
			},
		],
	},
];

export const router = createBrowserRouter(routes);
