import { createHashRouter } from "react-router";
import Clipboard from "@/pages/Clipboard";
import ContextMenu, { ContextSubmenu } from "@/pages/ContextMenu";
import Preference from "@/pages/Preference";
import Preview from "@/pages/Preview";

export const router = createHashRouter([
  {
    Component: Clipboard,
    path: "/",
  },
  {
    Component: Preference,
    path: "/preference",
  },
  {
    Component: ContextMenu,
    path: "/context-menu",
  },
  {
    Component: ContextSubmenu,
    path: "/context-submenu",
  },
  {
    Component: Preview,
    path: "/preview",
  },
]);
