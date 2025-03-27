import Main from "@/pages/Main";
import Preference from "@/pages/Preference";
import { createHashRouter } from "react-router-dom";

export const router = createHashRouter([
	{
		path: "/",
		Component: Main,
	},
	{
		path: "/preference",
		Component: Preference,
	},
]);
