import { createHashRouter } from "react-router";
import Main from "@/pages/Main";
import Preference from "@/pages/Preference";

export const router = createHashRouter([
  {
    Component: Main,
    path: "/",
  },
  {
    Component: Preference,
    path: "/preference",
  },
]);
