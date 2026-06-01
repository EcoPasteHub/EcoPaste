import type { FC, MouseEvent } from "react";
import { useSnapshot } from "valtio";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { clipboardViewState } from "@/stores/clipboardView";
import type { ClipboardGroup } from "@/types/clipboard";
import { cn } from "@/utils/cn";

interface GroupOption {
  label: string;
  value: ClipboardGroup;
  icon: string;
}

const GROUP_OPTIONS: GroupOption[] = [
  { icon: "i-lets-icons:widget", label: "全部", value: "all" },
  { icon: "i-lets-icons:file-dock", label: "文本", value: "text" },
  { icon: "i-lets-icons:img-box", label: "图片", value: "image" },
  { icon: "i-lets-icons:folder-file-alt", label: "文件", value: "files" },
  { icon: "i-lets-icons:star", label: "收藏", value: "favorite" },
];

/**
 * Header 下方的分组筛选栏：全部 / 文本 / 图片 / 文件 / 收藏，点击后写入 store，
 * List 监听 store 变化并重新查询。
 */
const Group: FC = () => {
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
      {GROUP_OPTIONS.map(({ label, value, icon }) => (
        <button
          className={cn(
            "flex cursor-pointer items-center gap-1 rounded-full px-2.5 py-1 text-xs transition-colors",
            {
              "bg-primary text-white": group === value,
              "text-secondary hover:bg-fill-tertiary": group !== value,
            },
          )}
          data-value={value}
          key={value}
          onClick={handleGroupClick}
          type="button"
        >
          <span className={cn(icon, "size-3.5")} />
          {label}
        </button>
      ))}
    </div>
  );
};

export default Group;
