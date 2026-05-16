import ReactDOM from "react-dom/client";
import App from "./App";
import "virtual:uno.css";
import "@unocss/reset/tailwind-compat.css";
import "./assets/css/global.scss";
import "mac-scrollbar/dist/mac-scrollbar.css";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <App />,
);
