import { Button } from "antd";
import { useState } from "react";
import { useSnapshot } from "valtio";
import KeyHint from "@/components/KeyHint";
import Popover from "@/components/Popover";
import Tooltip from "@/components/Tooltip";
import { clipboardStatsState } from "@/stores/clipboardStats";
import ShortcutList from "./ShortcutList";

/**
 * 剪贴板窗口底部条：左侧统计当前过滤下的总条数（由 List 写入共享 store，
 * Rust 列表查询附带返回），右侧展示窗口快捷键提示。
 */
const Footer = () => {
  const { total } = useSnapshot(clipboardStatsState);

  // Popover 打开时强制收起 Tooltip，避免两层浮层叠加遮挡。
  const [popoverOpen, setPopoverOpen] = useState(false);

  return (
    <div className="flex items-center justify-between p-3">
      <span className="text-text-tertiary text-xs">共 {total ?? 0} 项</span>

      <Popover
        content={<ShortcutList />}
        onOpenChange={setPopoverOpen}
        open={popoverOpen}
        title="快捷键"
        trigger="click"
      >
        <Tooltip open={popoverOpen ? false : void 0} title="快捷键">
          <Button
            icon={<KeyHint hintKey="K" iconName="i-lucide:keyboard" />}
            size="small"
            type="text"
          />
        </Tooltip>
      </Popover>
    </div>
  );
};

export default Footer;
