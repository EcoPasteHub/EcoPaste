import { emitTo } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useEventListener } from "ahooks";
import { type FC, Fragment, type MouseEvent, useState } from "react";
import { TAURI_EVENT } from "@/constants/events";
import { WINDOW_LABEL } from "@/constants/windows";
import { useTauriListen } from "@/hooks/useTauriListen";
import type { ClipboardAction } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import { formatShortcutDisplay } from "@/utils/shortcut";

/**
 * Rust `menu::context_window::ContextMenuShowPayload` 的前端镜像。
 * 字段语义见 `src-tauri/src/menu/context_window.rs`。
 */
interface ShowPayload {
  itemId: string;
  isFavorite: boolean;
  isPinned: boolean;
  groups: Array<
    Array<{
      action: ClipboardAction;
      label: string;
      accelerator: string | null;
      groups?: Array<{
        checked: boolean;
        id: string;
        label: string;
      }>;
    }>
  >;
}

/**
 * Windows 自定义右键菜单页（路由 `/context-menu`，仅在 Windows 由 Rust 端建窗加载）。
 * 接 `context-menu://show` 事件渲染分组菜单，点击项把动作 emit 回主窗 + 自己 hide。
 * 主窗的 `List.tsx` 已订阅 `clipboard://menu-action`，业务派发与 macOS 原生菜单走同一路径。
 */
const ContextMenu: FC = () => {
  const [payload, setPayload] = useState<ShowPayload | null>(null);

  useTauriListen<ShowPayload>(TAURI_EVENT.CONTEXT_MENU_SHOW, (event) => {
    setPayload(event.payload);
  });

  /**
   * ESC 关闭。`focusable: false` 的窗口 OS 焦点不在 webview 上，
   * 但 webview 内部 keydown 仍能收到 web 事件，足以支撑 ESC。
   */
  useEventListener("keydown", (event) => {
    if (event.key !== "Escape") return;

    void getCurrentWebviewWindow().hide();
  });

  const handlePick = async (action: ClipboardAction, groupId?: string) => {
    if (!payload) return;

    const eventPayload = groupId
      ? { action, groupId, itemId: payload.itemId }
      : { action, itemId: payload.itemId };

    await emitTo(
      WINDOW_LABEL.MAIN,
      TAURI_EVENT.CLIPBOARD_MENU_ACTION,
      eventPayload,
    );

    await getCurrentWebviewWindow().hide();
  };

  if (!payload) return null;

  return (
    <div className="w-55 select-none rounded-2 bg-ant-elevated p-1 shadow-lg">
      {payload.groups.map((group, gi) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: 分组顺序由后端写死，索引稳定。
        <Fragment key={gi}>
          {gi > 0 && <div className="my-1 h-px bg-ant-split" />}
          {group.map((item) => (
            <ContextMenuItem
              accelerator={item.accelerator}
              action={item.action}
              groups={item.groups ?? []}
              isDanger={item.action === "delete"}
              key={item.action}
              label={item.label}
              onPick={handlePick}
            />
          ))}
        </Fragment>
      ))}
    </div>
  );
};

interface ContextMenuItemProps {
  action: ClipboardAction;
  label: string;
  accelerator: string | null;
  groups: Array<{
    checked: boolean;
    id: string;
    label: string;
  }>;
  isDanger: boolean;
  onPick: (action: ClipboardAction, groupId?: string) => void;
}

const ContextMenuItem: FC<ContextMenuItemProps> = (props) => {
  const { action, label, accelerator, groups, isDanger, onPick } = props;
  const hasGroups = groups.length > 0;

  const handleClick = () => {
    if (hasGroups) return;

    onPick(action);
  };

  const handleGroupClick = (event: MouseEvent<HTMLButtonElement>) => {
    const groupId = event.currentTarget.dataset.groupId;
    if (!groupId) return;

    onPick(action, groupId);
  };

  return (
    <div className="group relative">
      <button
        className={cn(
          "flex h-8 w-full cursor-pointer items-center justify-between rounded-1.5 border-0 bg-transparent px-3 text-sm transition-colors",
          {
            "hover:(text-ant-light-solid bg-ant-error) text-ant-error":
              isDanger,
            "hover:bg-ant-text-hover": !isDanger,
          },
        )}
        onClick={handleClick}
        type="button"
      >
        <span className="truncate">{label}</span>
        {hasGroups ? (
          <span className="i-lucide:chevron-right ml-3 size-4 text-ant-description" />
        ) : (
          accelerator && (
            <span className="ml-3 whitespace-nowrap text-ant-description text-xs">
              {formatShortcutDisplay(accelerator)}
            </span>
          )
        )}
      </button>

      {hasGroups && (
        <div className="invisible absolute top-0 left-full w-55 rounded-2 bg-ant-elevated p-1 opacity-0 shadow-lg transition-opacity group-hover:visible group-hover:opacity-100">
          {groups.map((group) => (
            <button
              className="flex h-8 w-full cursor-pointer items-center gap-2 rounded-1.5 border-0 bg-transparent px-3 text-left text-sm transition-colors hover:bg-ant-text-hover"
              data-group-id={group.id}
              key={group.id}
              onClick={handleGroupClick}
              type="button"
            >
              <span
                className={cn("size-4 text-ant-primary", {
                  "i-lucide:check": group.checked,
                })}
              />
              <span className="truncate">{group.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default ContextMenu;
