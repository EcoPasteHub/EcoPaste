import { Button, Empty, Tooltip } from "antd";
import KeyHint from "@/components/KeyHint";
import SearchInput from "@/components/SearchInput";
import { cn } from "@/utils/cn";
import { isMac } from "@/utils/is";

const Clipboard = () => {
  return (
    <div
      className={cn("flex h-screen flex-col bg-container p-2", {
        "rounded-4": isMac,
      })}
      data-tauri-drag-region
    >
      <div className="flex items-center justify-between" data-tauri-drag-region>
        <img alt="logo" className="size-5" src="/logo.png" />

        <div className="flex items-center gap-1">
          <SearchInput
            className="w-40"
            placeholder="搜索剪贴板..."
            size="small"
          />

          <Tooltip title="固定窗口">
            <Button
              icon={
                <KeyHint hintKey="P">
                  <i className="i-lets-icons:pin text-4" />
                </KeyHint>
              }
              size="small"
              type="text"
            />
          </Tooltip>

          <Tooltip title="更多操作">
            <Button
              icon={<i className="i-lets-icons:meatballs-menu text-4" />}
              size="small"
              type="text"
            />
          </Tooltip>
        </div>
      </div>

      <div
        className="flex flex-1 flex-col items-center justify-center"
        data-tauri-drag-region
      >
        <Empty
          description="暂无剪贴板历史"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>

      <div className="flex items-center justify-between">
        <div>总项</div>

        <div>预设的窗口快捷键</div>
      </div>
    </div>
  );
};

export default Clipboard;
