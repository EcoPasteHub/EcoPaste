import { createHashRouter } from "react-router";
import Clipboard from "@/pages/Clipboard";
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
]);
