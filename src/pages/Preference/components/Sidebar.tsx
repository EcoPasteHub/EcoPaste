import { Card } from "antd";

/**
 * 偏好设置左侧分类导航：列出可切换的设置面板。
 */
const Sidebar = () => {
  return (
    <div className="h-full p-2">
      <Card
        classNames={{
          body: "h-full w-30 flex flex-col items-center gap-4 overflow-auto bg-linear-to-b from-blue-1 to-black/1 dark:bg-none",
          root: "h-full",
        }}
        variant="borderless"
      >
        111
      </Card>
    </div>
  );
};

export default Sidebar;
