import { createHashRouter } from "react-router";
import Clipboard from "@/pages/Clipboard";
import ContextMenu from "@/pages/ContextMenu";
import Preference from "@/pages/Preference";

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
]);
