import type { FC, MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { clipboardViewState } from "@/stores/clipboardView";
import type { ClipboardGroup } from "@/types/clipboard";
import { cn } from "@/utils/cn";

interface GroupOption {
  labelKey: string;
  value: ClipboardGroup;
  icon: string;
}

const GROUP_OPTIONS: GroupOption[] = [
  { icon: "i-lets-icons:widget", labelKey: "groups.all", value: "all" },
  { icon: "i-lets-icons:file-dock", labelKey: "groups.text", value: "text" },
  { icon: "i-lets-icons:img-box", labelKey: "groups.image", value: "image" },
  {
    icon: "i-lets-icons:folder-file-alt",
    labelKey: "groups.files",
    value: "files",
  },
  {
    icon: "i-lets-icons:star",
    labelKey: "groups.favorite",
    value: "favorite",
  },
];

/**
 * Header 下方的分组筛选栏：全部 / 文本 / 图片 / 文件 / 收藏，点击后写入 store，
 * List 监听 store 变化并重新查询。
 */
const Group: FC = () => {
  const { t } = useTranslation("clipboard");
  const { group } = useSnapshot(clipboardViewState);

  const handleGroupClick = (e: MouseEvent<HTMLButtonElement>) => {
    const value = e.currentTarget.dataset.value as ClipboardGroup;

    clipboardViewState.group = value;
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key !== "Tab") return;

    e.preventDefault();

    const values = GROUP_OPTIONS.map((o) => o.value);
    const current = values.indexOf(clipboardViewState.group);
    const next = e.shiftKey
      ? (current - 1 + values.length) % values.length
      : (current + 1) % values.length;

    clipboardViewState.group = values[next];
  };

  useKeyboardEvent("keydown", handleKeyDown);

  return (
    <div className="flex items-center gap-1 px-3 pb-2" data-tauri-drag-region>
      {GROUP_OPTIONS.map(({ labelKey, value, icon }) => (
        <button
          className={cn(
            "flex cursor-pointer items-center gap-1 rounded-full px-2.5 py-1 text-xs transition-colors",
            {
              "bg-ant-primary text-ant-light-solid": group === value,
              "text-ant-secondary hover:bg-ant-fill-tertiary": group !== value,
            },
          )}
          data-value={value}
          key={value}
          onClick={handleGroupClick}
          type="button"
        >
          <span className={cn(icon, "size-3.5")} />
          {t(labelKey)}
        </button>
      ))}
    </div>
  );
};

export default Group;
