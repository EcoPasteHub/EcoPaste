import { emitTo } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useEventListener } from "ahooks";
import { type FC, Fragment, useState } from "react";
import { TAURI_EVENT } from "@/constants/events";
import { WINDOW_LABEL } from "@/constants/windows";
import { useTauriListen } from "@/hooks/useTauriListen";
import type { ClipboardAction } from "@/types/clipboard";
import { cn } from "@/utils/cn";

/**
 * Rust `menu::context_window::ContextMenuShowPayload` 的前端镜像。
 * 字段语义见 `src-tauri/src/menu/context_window.rs`。
 */
interface ShowPayload {
  itemId: string;
  isFavorite: boolean;
  groups: Array<
    Array<{
      action: ClipboardAction;
      label: string;
      accelerator: string | null;
    }>
  >;
}

/**
 * 把 Tauri 加速键写法 `"CmdOrCtrl+Enter"` 翻译为 Windows 文案 `"Ctrl + Enter"`。
 * 本窗口只在 Windows 出现，直接硬编码 Ctrl。
 */
const formatAccelerator = (raw: string) => {
  return raw
    .replace(/CmdOrCtrl/g, "Ctrl")
    .replace(/Cmd/g, "Ctrl")
    .replace(/Backspace/g, "⌫")
    .replace(/\+/g, " + ");
};

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

  const handlePick = async (action: ClipboardAction) => {
    if (!payload) return;

    await emitTo(WINDOW_LABEL.MAIN, TAURI_EVENT.CLIPBOARD_MENU_ACTION, {
      action,
      itemId: payload.itemId,
    });

    await getCurrentWebviewWindow().hide();
  };

  if (!payload) return null;

  return (
    <div className="w-full select-none rounded-2 bg-elevated px-1 py-1 shadow-lg">
      {payload.groups.map((group, gi) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: 分组顺序由后端写死，索引稳定。
        <Fragment key={gi}>
          {gi > 0 && <div className="my-1 h-px bg-border" />}
          {group.map((item) => (
            <ContextMenuItem
              accelerator={item.accelerator}
              action={item.action}
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
  isDanger: boolean;
  onPick: (action: ClipboardAction) => void;
}

const ContextMenuItem: FC<ContextMenuItemProps> = (props) => {
  const { action, label, accelerator, isDanger, onPick } = props;

  const handleClick = () => {
    onPick(action);
  };

  return (
    <button
      className={cn(
        "flex h-8 w-full cursor-default items-center justify-between rounded-1 border-0 bg-transparent px-2 text-sm hover:bg-fill",
        {
          "text-error": isDanger,
          "text-text-primary": !isDanger,
        },
      )}
      onClick={handleClick}
      type="button"
    >
      <span className="truncate">{label}</span>
      {accelerator && (
        <span className="ml-2 whitespace-nowrap text-text-tertiary text-xs">
          {formatAccelerator(accelerator)}
        </span>
      )}
    </button>
  );
};

export default ContextMenu;
