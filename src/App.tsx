import { ConfigProvider } from "antd";
import { use } from "react";
import { RouterProvider } from "react-router";
import { router } from "./router";
import { settingsReady } from "./stores/settings";

/**
 * 等待 Rust 设置首屏快照灌入后再渲染，避免组件读到空对象闪烁默认值。
 * `use()` 在 promise pending 时抛出，由父级（`main.tsx`）的 Suspense 接住。
 */
const App = () => {
  use(settingsReady);

  return (
    <ConfigProvider>
      <RouterProvider router={router} />
    </ConfigProvider>
  );
};

export default App;
