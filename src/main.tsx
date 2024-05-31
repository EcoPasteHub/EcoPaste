import ReactDOM from "react-dom/client";
import App from "./App";
import "virtual:uno.css";
import "@unocss/reset/tailwind-compat.css";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
	<App />,
);
