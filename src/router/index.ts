// @unocss-include
import PreferenceLayout from "@/layouts/Preference";
import ClipboardPanel from "@/pages/Clipboard/Panel";
import { type RouteObject, createHashRouter } from "react-router-dom";

export const routes: RouteObject[] = [
	{
		path: "/",
		Component: ClipboardPanel,
	},
	{
		path: "/preference",
		Component: PreferenceLayout,
	},
];

export const router = createHashRouter(routes);
