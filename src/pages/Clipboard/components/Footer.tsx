import { Button, Popover, Tooltip } from "antd";
import KeyHint from "@/components/KeyHint";
import { useClipboardTotal } from "@/hooks/useClipboardTotal";

/**
 * 剪贴板窗口底部条：左侧统计总项数，右侧展示窗口快捷键提示。
 */
const Footer = () => {
  const total = useClipboardTotal();

  return (
    <div className="flex items-center justify-between p-3">
      <span className="text-text-tertiary text-xs">共 {total} 项</span>

      <Popover title="快捷键" trigger="click">
        <Tooltip placement="left" title="快捷键">
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
