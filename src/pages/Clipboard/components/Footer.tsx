import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import CustomIconButton from "@/components/CustomIconButton";
import KeyHint from "@/components/KeyHint";
import Popover from "@/components/Popover";
import { clipboardStatsState } from "@/stores/clipboardStats";
import { settingsState } from "@/stores/settings";
import { usesClipboardSheetLayout } from "../layout";
import ShortcutList from "./ShortcutList";

/**
 * 剪贴板窗口底部条：左侧统计当前过滤下的总条数（由 List 写入共享 store，
 * Rust 列表查询附带返回），右侧展示窗口快捷键提示。
 */
const Footer = () => {
  const { t } = useTranslation("clipboard");
  const { total } = useSnapshot(clipboardStatsState);
  const settings = useSnapshot(settingsState);
  const isSheetLayout = usesClipboardSheetLayout(
    settings.clipboard.window.position,
  );

  // Popover 打开时强制收起 Tooltip，避免两层浮层叠加遮挡。
  const [popoverOpen, setPopoverOpen] = useState(false);

  const handleShortcutKeyPress = () => {
    setPopoverOpen((prev) => {
      return !prev;
    });
  };

  if (isSheetLayout) return null;

  return (
    <div className="flex items-center justify-between px-3 py-1">
      <span className="text-ant-tertiary text-xs">
        {t("footer.total", { count: total ?? 0 })}
      </span>

      <Popover
        content={<ShortcutList />}
        onOpenChange={setPopoverOpen}
        open={popoverOpen}
        title={t("footer.shortcuts")}
        tooltip={t("footer.shortcuts")}
        trigger="click"
      >
        <CustomIconButton
          icon={
            <KeyHint
              hintKey="K"
              iconName="i-lucide:keyboard"
              onKeyPress={handleShortcutKeyPress}
            />
          }
          size="small"
          type="text"
        />
      </Popover>
    </div>
  );
};

export default Footer;
