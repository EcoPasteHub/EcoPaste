import { useEventListener } from "ahooks";
import { App as AntdApp, ConfigProvider } from "antd";
import { use } from "react";
import { RouterProvider } from "react-router";
import { router } from "./router";
import { settingsReady } from "./stores/settings";
import { log } from "./utils/log";

/**
 * 等待 Rust 设置首屏快照灌入后再渲染，避免组件读到空对象闪烁默认值。
 * `use()` 在 promise pending 时抛出，由父级（`main.tsx`）的 Suspense 接住。
 */
const App = () => {
  use(settingsReady);

  // 兜底未捕获的 Promise rejection：统一进日志通道，避免只在 devtools 红字闪过、生产环境完全无痕。
  useEventListener("unhandledrejection", (event) => {
    const { reason } = event;

    log.error(
      "unhandled promise rejection",
      reason instanceof Error ? reason : { reason },
    );
  });

  // 兜底未捕获的同步异常（含资源加载错误）。React 渲染错误由 ErrorBoundary 接，不会走到这里。
  useEventListener("error", (event) => {
    const { error, ...rest } = event;

    log.error("uncaught error", error instanceof Error ? error : rest);
  });

  return (
    <ConfigProvider>
      <AntdApp>
        <RouterProvider router={router} />
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;
