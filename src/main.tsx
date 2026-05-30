import ReactDOM from "react-dom/client";
import App from "@/App";
import { loadSettings } from "@/stores/settings";
import "@/styles/index.css";

// 阻塞首屏：拿到设置后再渲染。Rust 侧 load_from_disk 已有 .bak/Default 兜底，
// 所以这里只对 IPC 通信失败这种异常路径打点，组件层通过 settingsState.loaded 兜底。
const root = ReactDOM.createRoot(
  document.getElementById("root") as HTMLElement,
);

loadSettings()
  .catch((err) => {
    // biome-ignore lint/suspicious/noConsole: 启动期日志通道未就绪，只能落控制台
    console.error("load settings failed", err);
  })
  .finally(() => {
    root.render(<App />);
  });
