import { createHashRouter } from "react-router";
import Clipboard from "@/pages/Clipboard";
import ContextMenu, { ContextSubmenu } from "@/pages/ContextMenu";
import Onboarding from "@/pages/Onboarding";
import Preference from "@/pages/Preference";
import Preview from "@/pages/Preview";
import Update from "@/pages/Update";

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
    Component: Onboarding,
    path: "/onboarding",
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
  {
    Component: Update,
    path: "/update",
  },
]);
