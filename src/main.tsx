import ReactDOM from "react-dom/client";
import App from "@/App";
import { loadSettings } from "@/stores/settings";
import { log } from "@/utils/log";
import "@/styles/index.css";

// 阻塞首屏：拿到设置后再渲染。Rust 侧 load_from_disk 已有 .bak/Default 兜底，
// 这里只对 IPC 通信失败这种异常路径打点，组件层通过 settingsState.loaded 兜底。
const root = ReactDOM.createRoot(
  document.getElementById("root") as HTMLElement,
);

loadSettings()
  .catch((err) => {
    log.error("load settings failed", err);
  })
  .finally(() => {
    root.render(<App />);
  });
