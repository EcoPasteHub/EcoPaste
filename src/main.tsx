import { Suspense } from "react";
import ReactDOM from "react-dom/client";
import App from "./App";

import "virtual:uno.css";
import "./styles/global.scss";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <Suspense fallback={null}>
    <App />
  </Suspense>,
);
