import { ConfigProvider } from "antd";
import { RouterProvider } from "react-router";
import { router } from "./router";

const App = () => {
  return (
    <ConfigProvider>
      <RouterProvider router={router} />
    </ConfigProvider>
  );
};

export default App;
