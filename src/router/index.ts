import { createHashRouter } from "react-router-dom";
import Main from "@/pages/Main";
import OCR from "@/pages/OCR";
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
  {
    Component: OCR,
    path: "/ocr",
  },
]);
