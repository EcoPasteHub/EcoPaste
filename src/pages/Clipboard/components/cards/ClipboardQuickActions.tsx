import { useUnmount } from "ahooks";
import type { FC, MouseEvent, SyntheticEvent } from "react";
import { useRef, useState } from "react";
import Tooltip from "@/components/Tooltip";
import {
  filterAvailableItemActions,
  type ItemActionLabels,
  isCopyItemAction,
  resolveItemActionPresentation,
} from "@/constants/itemActions";
import type { ClipboardItem } from "@/types/clipboard";
import type { ItemAction } from "@/types/settings";
import { cn } from "@/utils/cn";

interface ClipboardQuickActionsProps {
  item: ClipboardItem;
  labels?: ItemActionLabels;
  onQuickAction?: (action: ItemAction) => Promise<void> | void;
  quickActions: ItemAction[];
  visible: boolean;
}

interface QuickActionButtonProps {
  action: ItemAction;
  isFavorite: boolean;
  labels: ItemActionLabels;
  onQuickAction: (action: ItemAction) => Promise<void> | void;
  tabIndex: 0 | -1;
}

/**
 * 在卡片 meta 右侧展示时间，并在 hover 时替换为当前条目可执行的快捷动作。
 */
const ClipboardQuickActions: FC<ClipboardQuickActionsProps> = (props) => {
  const { item, labels, onQuickAction, quickActions, visible } = props;
  const availableActions = filterAvailableItemActions(quickActions, item);
  const enabled =
    availableActions.length > 0 && Boolean(labels && onQuickAction);
  const actionsVisible = visible && enabled;
  const tabIndex = actionsVisible ? 0 : -1;

  return (
    <div className="grid h-6 shrink-0 items-center justify-items-end overflow-hidden">
      <span
        className={cn(
          "col-start-1 row-start-1 transition-all duration-150 ease-out motion-reduce:transition-none",
          {
            "-translate-y-0.5 opacity-0": actionsVisible,
          },
        )}
      >
        {item.displayCreatedAt ?? item.createdAt}
      </span>

      {enabled && onQuickAction && labels ? (
        <div
          aria-hidden={!actionsVisible}
          className={cn(
            "pointer-events-none col-start-1 row-start-1 flex translate-y-0.5 items-center gap-0.5 opacity-0 transition-all duration-150 ease-out motion-reduce:transition-none",
            {
              "pointer-events-auto translate-y-0 opacity-100": actionsVisible,
            },
          )}
        >
          {availableActions.map((action) => {
            return (
              <QuickActionButton
                action={action}
                isFavorite={item.isFavorite}
                key={action}
                labels={labels}
                onQuickAction={onQuickAction}
                tabIndex={tabIndex}
              />
            );
          })}
        </div>
      ) : null}
    </div>
  );
};

export default ClipboardQuickActions;

/**
 * 单个 hover 快捷动作按钮；按下时阻止事件冒泡，避免触发卡片点击或自动粘贴。
 */
const QuickActionButton: FC<QuickActionButtonProps> = (props) => {
  const { action, isFavorite, labels, onQuickAction, tabIndex } = props;
  const [copied, setCopied] = useState(false);
  const resetTimerRef = useRef<number | null>(null);
  const activeFavorite = action === "star" && isFavorite;
  const copyAction = isCopyItemAction(action);
  const presentation = resolveItemActionPresentation(action, labels, {
    copied,
    isFavorite: activeFavorite,
  });

  const stopQuickActionEvent = (event: SyntheticEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const clearCopiedResetTimer = () => {
    if (resetTimerRef.current === null) return;

    window.clearTimeout(resetTimerRef.current);
    resetTimerRef.current = null;
  };

  const startCopiedFeedback = () => {
    setCopied(true);
    clearCopiedResetTimer();
    resetTimerRef.current = window.setTimeout(() => {
      setCopied(false);
      resetTimerRef.current = null;
    }, 1000);
  };

  useUnmount(() => {
    clearCopiedResetTimer();
  });

  const handleClick = async (event: MouseEvent<HTMLButtonElement>) => {
    stopQuickActionEvent(event);

    if (copyAction && copied) return;

    await onQuickAction(action);

    if (!copyAction) return;

    startCopiedFeedback();
  };

  return (
    <Tooltip title={presentation.label}>
      <button
        aria-label={presentation.label}
        className={cn(
          "flex size-5 items-center justify-center rounded-1.5 border-0 bg-transparent text-ant-secondary transition-colors hover:bg-ant-fill-tertiary hover:text-ant-text motion-reduce:transition-none",
          {
            "text-ant-error hover:text-ant-error": presentation.danger,
            "text-ant-success hover:text-ant-success": copied,
            "text-ant-warning hover:text-ant-warning": activeFavorite,
          },
        )}
        onAuxClick={stopQuickActionEvent}
        onClick={handleClick}
        onContextMenu={stopQuickActionEvent}
        onDoubleClick={stopQuickActionEvent}
        onMouseDown={stopQuickActionEvent}
        onPointerDown={stopQuickActionEvent}
        tabIndex={tabIndex}
        type="button"
      >
        <i aria-hidden="true" className={cn(presentation.icon, "text-sm")} />
      </button>
    </Tooltip>
  );
};
