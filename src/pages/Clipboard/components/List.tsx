import { Empty } from "antd";

/**
 * 剪贴板历史列表区：展示剪贴板记录，暂无数据时显示空状态。
 */
const List = () => {
  return (
    <div
      className="flex flex-1 flex-col items-center justify-center"
      data-tauri-drag-region
    >
      <Empty
        description="暂无剪贴板历史"
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      />
    </div>
  );
};

export default List;
