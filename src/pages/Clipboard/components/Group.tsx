import { useMount } from "ahooks";
import { Modal } from "antd";
import type { TFunction } from "i18next";
import type {
  Dispatch,
  FC,
  MouseEvent,
  RefObject,
  SetStateAction,
} from "react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import {
  createClipboardGroup,
  deleteClipboardGroup,
  listClipboardGroups,
  updateClipboardGroup,
} from "@/commands";
import ClipboardGroupIcon from "@/components/ClipboardGroupIcon";
import ClipboardGroupModal from "@/components/ClipboardGroupModal";
import Dropdown, { type DropdownMenuItems } from "@/components/Dropdown";
import KeyHint from "@/components/KeyHint";
import Tooltip from "@/components/Tooltip";
import { TAURI_EVENT } from "@/constants/events";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { useTauriListen } from "@/hooks/useTauriListen";
import { clipboardViewState } from "@/stores/clipboardView";
import type {
  ClipboardCategory,
  ClipboardGroupIcon as ClipboardGroupIconValue,
  ClipboardGroupInput,
  ClipboardGroupRecord,
  ClipboardRange,
} from "@/types/clipboard";
import { cn } from "@/utils/cn";

type GroupModalMode = "create" | "edit";
type GroupMenuAction = "delete" | "edit" | "hide";

interface RangeGroupOption {
  labelKey: string;
  value: ClipboardRange;
  icon: ClipboardGroupIconValue;
}

interface CategoryGroupOption {
  labelKey: string;
  value: ClipboardCategory;
  icon: ClipboardGroupIconValue;
}

interface OverflowGroupMenuLabelProps {
  menuItems: DropdownMenuItems;
  onContext: (record: ClipboardGroupRecord) => void;
  onMenuClick: (info: { key: string }) => void;
  record: ClipboardGroupRecord;
}

const RANGE_GROUP_OPTIONS: RangeGroupOption[] = [
  { icon: "i-lets-icons:widget", labelKey: "groups.all", value: "all" },
  {
    icon: "i-lets-icons:star",
    labelKey: "groups.favorite",
    value: "favorite",
  },
];

const CATEGORY_GROUP_OPTIONS: CategoryGroupOption[] = [
  { icon: "i-lets-icons:file-dock", labelKey: "groups.text", value: "text" },
  { icon: "i-lets-icons:img-box", labelKey: "groups.image", value: "image" },
  {
    icon: "i-lets-icons:folder-file-alt",
    labelKey: "groups.files",
    value: "files",
  },
];

const GROUP_MENU_ACTION = {
  DELETE: "delete",
  EDIT: "edit",
  HIDE: "hide",
} as const satisfies Record<string, GroupMenuAction>;

const GROUP_BUTTON_BASE_CLASS =
  "flex size-6 shrink-0 cursor-pointer items-center justify-center rounded-1.5 border-0 bg-transparent p-0 transition-colors";
const GROUP_ICON_BUTTON_CLASS = GROUP_BUTTON_BASE_CLASS;
const GROUP_BUTTON_WIDTH = 24;
const GROUP_BUTTON_GAP = 4;

const MORE_MENU_NEW_GROUP_KEY = "newGroup";

/**
 * Header 下方的分组筛选栏：内置类型分组 + 自定义分组入口。
 */
const Group: FC = () => {
  const { t } = useTranslation(["clipboard", "common"]);
  const { category, groupId, range } = useSnapshot(clipboardViewState);

  const [customGroups, setCustomGroups] = useState<ClipboardGroupRecord[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [modalMode, setModalMode] = useState<GroupModalMode>("create");
  const [moreMenuOpen, setMoreMenuOpen] = useState(false);
  const [visibleCustomGroupCount, setVisibleCustomGroupCount] = useState(
    Number.POSITIVE_INFINITY,
  );
  const [editingGroup, setEditingGroup] = useState<ClipboardGroupRecord | null>(
    null,
  );
  const toolbarRef = useRef<HTMLDivElement>(null);
  const customGroupListRef = useRef<HTMLDivElement>(null);
  const contextGroupRef = useRef<ClipboardGroupRecord | null>(null);
  const deleteGroupRef = useRef<ClipboardGroupRecord | null>(null);

  const visibleCustomGroups = customGroups.filter((record) => {
    return !record.isHidden;
  });
  const inlineCustomGroups = visibleCustomGroups.slice(
    0,
    visibleCustomGroupCount,
  );
  const overflowCustomGroups = visibleCustomGroups.slice(
    visibleCustomGroupCount,
  );

  /**
   * 从 Rust 拉取自定义分组。
   */
  const loadGroups = async () => {
    const groups = await listClipboardGroups();

    setCustomGroups(groups);
    scheduleVisibleCustomGroupCountUpdate(
      toolbarRef,
      customGroupListRef,
      setVisibleCustomGroupCount,
      groups.filter((record) => {
        return !record.isHidden;
      }).length,
    );
    ensureSelectedGroupStillExists(groups);
  };

  /**
   * 首次挂载时拉取分组。
   */
  useMount(() => {
    void loadGroups();
  });

  /**
   * 其他窗口或命令修改分组后刷新本地列表。
   */
  const handleGroupsUpdated = () => {
    void loadGroups();
  };

  useTauriListen(TAURI_EVENT.CLIPBOARD_GROUPS_UPDATED, handleGroupsUpdated);

  /**
   * 容器尺寸变化时重新测量溢出状态。
   */
  useEffect(() => {
    const toolbar = toolbarRef.current;
    const customList = customGroupListRef.current;
    if (!toolbar || !customList) return;

    const updateVisibleCustomGroupCount = () => {
      commitVisibleCustomGroupCount(
        toolbar,
        customList,
        setVisibleCustomGroupCount,
        visibleCustomGroups.length,
      );
    };

    const observer = new ResizeObserver(updateVisibleCustomGroupCount);
    observer.observe(toolbar);
    updateVisibleCustomGroupCount();

    return () => {
      observer.disconnect();
    };
  }, [visibleCustomGroups.length]);

  /**
   * 切换范围；范围必须始终保留一个选中项。
   */
  const selectRange = (value: ClipboardRange) => {
    clipboardViewState.range = value;
  };

  /**
   * 切换分类；再次点击当前分类时取消。
   */
  const toggleCategory = (value: ClipboardCategory) => {
    clipboardViewState.category =
      clipboardViewState.category === value ? null : value;
  };

  /**
   * 切换到自定义分组；再次点击当前分组时取消。
   */
  const toggleCustomGroup = (id: string) => {
    clipboardViewState.groupId = clipboardViewState.groupId === id ? null : id;
  };

  /**
   * 点击分组按钮时根据 data 属性切换筛选。
   */
  const handleGroupClick = (event: MouseEvent<HTMLButtonElement>) => {
    const type = event.currentTarget.dataset.type;
    const value = event.currentTarget.dataset.value;
    const nextGroupId = event.currentTarget.dataset.groupId;

    if (nextGroupId) {
      toggleCustomGroup(nextGroupId);
      return;
    }

    if (type === "range" && isRangeGroup(value)) {
      selectRange(value);
      return;
    }

    if (type === "category" && isCategoryGroup(value)) {
      toggleCategory(value);
    }
  };

  /**
   * 记录右键菜单所属分组。
   */
  const handleCustomGroupContextMenu = (
    event: MouseEvent<HTMLButtonElement>,
  ) => {
    const nextGroupId = event.currentTarget.dataset.groupId;
    if (!nextGroupId) return;

    contextGroupRef.current =
      customGroups.find((record) => {
        return record.id === nextGroupId;
      }) ?? null;
  };

  /**
   * 处理分组栏快捷键：Cmd/Ctrl+Q 切换范围，左右键切分类，Tab / Shift+Tab 仅在可见自定义分组间循环。
   */
  const handleKeyDown = (event: KeyboardEvent) => {
    const eventModifierPressed = event.metaKey || event.ctrlKey;

    if (eventModifierPressed && event.key.toLowerCase() === "q") {
      event.preventDefault();
      toggleRange();

      return;
    }

    if (
      (event.key === "ArrowLeft" || event.key === "ArrowRight") &&
      !shouldUseNativeHorizontalNavigation(event)
    ) {
      event.preventDefault();
      selectAdjacentCategory(event.key === "ArrowLeft" ? -1 : 1);

      return;
    }

    if (event.key !== "Tab") return;

    event.preventDefault();

    const nextGroupId = selectAdjacentCustomGroup(
      visibleCustomGroups,
      groupId,
      event.shiftKey,
    );

    if (!nextGroupId) return;

    toggleCustomGroup(nextGroupId);
  };

  useKeyboardEvent("keydown", handleKeyDown);

  /**
   * 在全部 / 收藏范围之间循环切换，不影响分类与自定义分组筛选。
   */
  const toggleRange = () => {
    clipboardViewState.range =
      clipboardViewState.range === "all" ? "favorite" : "all";
  };

  /**
   * 按方向键在固定分类序列内循环；未选分类时从方向对应的端点进入。
   */
  const selectAdjacentCategory = (direction: -1 | 1) => {
    const options = CATEGORY_GROUP_OPTIONS.map((option) => {
      return option.value;
    });
    const currentCategory = clipboardViewState.category;
    const current = currentCategory ? options.indexOf(currentCategory) : -1;
    const startIndex = direction === 1 ? -1 : options.length;
    const nextIndex =
      (current === -1 ? startIndex + direction : current + direction) %
      options.length;
    const normalizedIndex = (nextIndex + options.length) % options.length;

    clipboardViewState.category = options[normalizedIndex];
  };

  /**
   * 打开新增分组弹框。
   */
  const openCreateModal = () => {
    setModalMode("create");
    setEditingGroup(null);
    setModalOpen(true);
  };

  /**
   * 打开编辑分组弹框。
   */
  const openEditModal = (record: ClipboardGroupRecord) => {
    setModalMode("edit");
    setEditingGroup(record);
    setModalOpen(true);
  };

  /**
   * 关闭新增 / 编辑分组弹框。
   */
  const closeModal = () => {
    setModalOpen(false);
    setEditingGroup(null);
  };

  /**
   * 保存分组弹框内容。
   */
  const handleModalSubmit = async (input: ClipboardGroupInput) => {
    if (modalMode === "create") {
      await createClipboardGroup(input);
      closeModal();
      return;
    }

    if (!editingGroup) return;

    await updateClipboardGroup(editingGroup.id, input);
    closeModal();
  };

  /**
   * 执行自定义分组右键菜单动作。
   */
  const handleGroupMenuClick = (info: { key: string }) => {
    const record = contextGroupRef.current;
    if (!record) return;

    if (info.key === GROUP_MENU_ACTION.EDIT) {
      openEditModal(record);
      return;
    }

    if (info.key === GROUP_MENU_ACTION.HIDE) {
      void updateClipboardGroup(record.id, {
        icon: record.icon,
        isHidden: true,
        name: record.name,
      });
      return;
    }

    if (info.key === GROUP_MENU_ACTION.DELETE) {
      requestDeleteGroup(record);
    }
  };

  /**
   * 执行更多菜单动作。
   */
  const handleMoreMenuClick = (info: { key: string }) => {
    setMoreMenuOpen(false);

    if (info.key === MORE_MENU_NEW_GROUP_KEY) {
      openCreateModal();
      return;
    }

    const id = parseMoreMenuGroupId(info.key);
    if (!id) return;

    toggleCustomGroup(id);
  };

  /**
   * 弹出删除确认框。
   */
  const requestDeleteGroup = (record: ClipboardGroupRecord) => {
    deleteGroupRef.current = record;

    Modal.confirm({
      content: record.name,
      okButtonProps: { danger: true },
      okText: t("common:actions.delete"),
      onOk: confirmDeleteGroup,
      title: t("clipboard:groups.delete"),
    });
  };

  /**
   * 确认删除当前待删除分组。
   */
  const confirmDeleteGroup = async () => {
    const record = deleteGroupRef.current;
    if (!record) return;

    await deleteClipboardGroup(record.id);

    if (clipboardViewState.groupId === record.id) {
      clipboardViewState.groupId = null;
    }

    deleteGroupRef.current = null;
  };

  const groupMenuItems: DropdownMenuItems = [
    {
      icon: "i-lucide:pencil",
      key: GROUP_MENU_ACTION.EDIT,
      label: t("clipboard:groups.edit"),
    },
    {
      icon: "i-lucide:eye-off",
      key: GROUP_MENU_ACTION.HIDE,
      label: t("clipboard:groups.hide"),
    },
    { type: "divider" },
    {
      danger: true,
      icon: "i-lucide:trash-2",
      key: GROUP_MENU_ACTION.DELETE,
      label: t("clipboard:groups.delete"),
    },
  ];

  /**
   * 记录溢出菜单中右键菜单所属分组。
   */
  const handleOverflowGroupContext = (record: ClipboardGroupRecord) => {
    contextGroupRef.current = record;
  };

  const moreMenuItems = buildMoreMenuItems(
    overflowCustomGroups,
    groupMenuItems,
    handleGroupMenuClick,
    handleOverflowGroupContext,
    t,
  );
  const moreMenuSelectedKeys = groupId ? [buildMoreMenuGroupKey(groupId)] : [];
  const moreButtonSelected = overflowCustomGroups.some((record) => {
    return record.id === groupId;
  });

  /**
   * 同步更多菜单展开状态，用于打开菜单时压掉按钮 Tooltip。
   */
  const handleMoreOpenChange = (open: boolean) => {
    setMoreMenuOpen(open);
  };

  /**
   * 渲染溢出分组菜单按钮。
   */
  const renderMoreButton = () => {
    if (overflowCustomGroups.length === 0) return null;

    return (
      <Dropdown
        menu={{
          items: moreMenuItems,
          onClick: handleMoreMenuClick,
          selectedKeys: moreMenuSelectedKeys,
        }}
        onOpenChange={handleMoreOpenChange}
        open={moreMenuOpen}
        trigger={["click"]}
      >
        <Tooltip
          open={moreMenuOpen ? false : void 0}
          title={t("clipboard:groups.more")}
        >
          <button
            className={cn(GROUP_BUTTON_BASE_CLASS, {
              "bg-ant-primary text-ant-light-solid": moreButtonSelected,
              "text-ant-secondary hover:bg-ant-fill-tertiary":
                !moreButtonSelected,
            })}
            type="button"
          >
            <i aria-hidden className="i-lucide:more-horizontal text-sm!" />
          </button>
        </Tooltip>
      </Dropdown>
    );
  };

  /**
   * 渲染独立新增按钮；存在溢出菜单时由菜单内新增入口承接。
   */
  const renderCreateButton = () => {
    if (overflowCustomGroups.length > 0) return null;

    return (
      <Tooltip title={t("clipboard:groups.add")}>
        <button
          className={cn(
            GROUP_BUTTON_BASE_CLASS,
            "text-ant-secondary hover:bg-ant-fill-tertiary",
          )}
          onClick={openCreateModal}
          type="button"
        >
          <i aria-hidden className="i-lucide:plus text-sm!" />
        </button>
      </Tooltip>
    );
  };

  /**
   * 渲染范围按钮。
   */
  const renderRangeButton = ({ labelKey, value, icon }: RangeGroupOption) => {
    const selected = range === value;
    const nextRange =
      range === "all" ? "favorite" : range === "favorite" ? "all" : void 0;
    const showShortcutHint = nextRange === value;

    return renderFilterButton({
      icon,
      label: t(`clipboard:${labelKey}`),
      selected,
      showShortcutHint,
      type: "range",
      value,
    });
  };

  /**
   * 渲染分类按钮。
   */
  const renderCategoryButton = ({
    labelKey,
    value,
    icon,
  }: CategoryGroupOption) => {
    const selected = category === value;

    return renderFilterButton({
      icon,
      label: t(`clipboard:${labelKey}`),
      selected,
      type: "category",
      value,
    });
  };

  /**
   * 渲染单个筛选按钮。
   */
  const renderFilterButton = (options: {
    icon: ClipboardGroupIconValue;
    label: string;
    selected: boolean;
    showShortcutHint?: boolean;
    type: "category" | "range";
    value: ClipboardCategory | ClipboardRange;
  }) => {
    const { icon, label, selected, showShortcutHint, type, value } = options;

    return (
      <Tooltip key={`${type}:${value}`} title={label}>
        <button
          className={cn(GROUP_ICON_BUTTON_CLASS, {
            "bg-ant-primary text-ant-light-solid": selected,
            "text-ant-secondary hover:bg-ant-fill-tertiary": !selected,
          })}
          data-type={type}
          data-value={value}
          onClick={handleGroupClick}
          type="button"
        >
          {showShortcutHint ? (
            <KeyHint hintKey="Q">
              <ClipboardGroupIcon icon={icon} selected={selected} />
            </KeyHint>
          ) : (
            <ClipboardGroupIcon icon={icon} selected={selected} />
          )}
        </button>
      </Tooltip>
    );
  };

  return (
    <>
      <div
        className="flex items-center gap-1 overflow-hidden px-3 pb-2"
        data-tauri-drag-region
        ref={toolbarRef}
      >
        {RANGE_GROUP_OPTIONS.map(renderRangeButton)}
        <GroupSeparator />
        {CATEGORY_GROUP_OPTIONS.map(renderCategoryButton)}
        <GroupSeparator />

        <div
          className="flex min-w-0 shrink-0 items-center gap-1 overflow-hidden"
          ref={customGroupListRef}
        >
          {inlineCustomGroups.map((record) => {
            const selected = groupId === record.id;

            return (
              <Dropdown
                key={record.id}
                menu={{
                  items: groupMenuItems,
                  onClick: handleGroupMenuClick,
                }}
                trigger={["contextMenu"]}
              >
                <Tooltip title={record.name}>
                  <button
                    className={cn(GROUP_ICON_BUTTON_CLASS, {
                      "bg-ant-primary text-ant-light-solid": selected,
                      "text-ant-secondary hover:bg-ant-fill-tertiary":
                        !selected,
                    })}
                    data-group-id={record.id}
                    onClick={handleGroupClick}
                    onContextMenu={handleCustomGroupContextMenu}
                    type="button"
                  >
                    <ClipboardGroupIcon
                      icon={record.icon}
                      selected={selected}
                    />
                  </button>
                </Tooltip>
              </Dropdown>
            );
          })}
        </div>

        {renderMoreButton()}
        {renderCreateButton()}
      </div>

      <ClipboardGroupModal
        group={editingGroup}
        mode={modalMode}
        onCancel={closeModal}
        onSubmit={handleModalSubmit}
        open={modalOpen}
      />
    </>
  );
};

/**
 * 分隔范围、分类、自定义分组三段。
 */
const GroupSeparator: FC = () => {
  return <span aria-hidden className="mx-1 h-4 w-px shrink-0 bg-ant-split" />;
};

/**
 * 溢出菜单里的分组行：左键选择分组，右键打开同一套分组管理菜单。
 */
const OverflowGroupMenuLabel: FC<OverflowGroupMenuLabelProps> = (props) => {
  const { menuItems, onContext, onMenuClick, record } = props;

  const handleContextMenu = () => {
    onContext(record);
  };

  return (
    <Dropdown
      menu={{
        items: menuItems,
        onClick: onMenuClick,
      }}
      trigger={["contextMenu"]}
    >
      <span
        className="flex min-w-28 items-center gap-2"
        onContextMenu={handleContextMenu}
        role="menuitem"
        tabIndex={-1}
      >
        <ClipboardGroupIcon icon={record.icon} inheritColor />
        <span>{record.name}</span>
      </span>
    </Dropdown>
  );
};

/**
 * 下一帧提交可见自定义分组数量；分组数据更新后等待 DOM 渲染完成再测量。
 */
function scheduleVisibleCustomGroupCountUpdate(
  toolbarRef: RefObject<HTMLDivElement | null>,
  groupListRef: RefObject<HTMLDivElement | null>,
  setVisibleCustomGroupCount: Dispatch<SetStateAction<number>>,
  groupCount: number,
) {
  requestAnimationFrame(() => {
    commitVisibleCustomGroupCount(
      toolbarRef.current,
      groupListRef.current,
      setVisibleCustomGroupCount,
      groupCount,
    );
  });
}

/**
 * 根据自定义分组栏可用宽度写入可见分组数量。
 */
function commitVisibleCustomGroupCount(
  toolbar: HTMLDivElement | null,
  customList: HTMLDivElement | null,
  setVisibleCustomGroupCount: Dispatch<SetStateAction<number>>,
  groupCount: number,
) {
  const rawCapacity =
    toolbar && customList
      ? computeCustomGroupCapacity(toolbar, customList)
      : groupCount;
  const visibleCount = Math.min(groupCount, rawCapacity);

  setVisibleCustomGroupCount((current) => {
    if (current === visibleCount) return current;

    return visibleCount;
  });
}

/**
 * 按整条分组栏剩余宽度计算自定义分组可显示数量。
 */
function computeCustomGroupCapacity(
  toolbar: HTMLDivElement,
  customList: HTMLDivElement,
) {
  const toolbarRect = toolbar.getBoundingClientRect();
  const customRect = customList.getBoundingClientRect();
  const customStart = customRect.left - toolbarRect.left;
  const actionSlotWidth = GROUP_BUTTON_GAP + GROUP_BUTTON_WIDTH;
  const availableWidth = Math.max(
    0,
    toolbar.clientWidth - customStart - actionSlotWidth,
  );

  return Math.max(
    0,
    Math.floor(
      (availableWidth + GROUP_BUTTON_GAP) /
        (GROUP_BUTTON_WIDTH + GROUP_BUTTON_GAP),
    ),
  );
}

/**
 * 构建更多菜单项：新增入口 + 全部分组快速入口。
 */
function buildMoreMenuItems(
  groups: ClipboardGroupRecord[],
  groupMenuItems: DropdownMenuItems,
  onGroupMenuClick: (info: { key: string }) => void,
  onGroupContext: (record: ClipboardGroupRecord) => void,
  t: TFunction<["clipboard", "common"]>,
): DropdownMenuItems {
  const groupItems = groups.map((record) => {
    return {
      key: buildMoreMenuGroupKey(record.id),
      label: (
        <OverflowGroupMenuLabel
          menuItems={groupMenuItems}
          onContext={onGroupContext}
          onMenuClick={onGroupMenuClick}
          record={record}
        />
      ),
    };
  });

  if (groupItems.length === 0) {
    return [
      {
        icon: "i-lucide:plus",
        key: MORE_MENU_NEW_GROUP_KEY,
        label: t("clipboard:groups.add"),
      },
    ];
  }

  return [
    {
      icon: "i-lucide:plus",
      key: MORE_MENU_NEW_GROUP_KEY,
      label: t("clipboard:groups.add"),
    },
    { type: "divider" },
    ...groupItems,
  ];
}

/**
 * 生成更多菜单中的分组 key。
 */
function buildMoreMenuGroupKey(id: string) {
  return `group:${id}`;
}

/**
 * 从更多菜单 key 中解析自定义分组 id。
 */
function parseMoreMenuGroupId(key: string) {
  if (!key.startsWith("group:")) return null;

  return key.slice("group:".length);
}

/**
 * 判断字符串是否为范围分组值。
 */
function isRangeGroup(value: unknown): value is ClipboardRange {
  return RANGE_GROUP_OPTIONS.some((option) => {
    return option.value === value;
  });
}

/**
 * 判断字符串是否为分类分组值。
 */
function isCategoryGroup(value: unknown): value is ClipboardCategory {
  return CATEGORY_GROUP_OPTIONS.some((option) => {
    return option.value === value;
  });
}

/**
 * 在可见自定义分组间前后循环；当前未选中分组时，正向取第一个，反向取最后一个。
 */
function selectAdjacentCustomGroup(
  groups: ClipboardGroupRecord[],
  groupId: string | null,
  reverse: boolean,
) {
  if (groups.length === 0) return null;

  const currentIndex = groupId
    ? groups.findIndex((record) => {
        return record.id === groupId;
      })
    : -1;

  if (reverse) {
    if (currentIndex === -1) return groups[groups.length - 1]?.id ?? null;

    return (
      groups[(currentIndex - 1 + groups.length) % groups.length]?.id ?? null
    );
  }

  if (currentIndex === -1) return groups[0]?.id ?? null;

  return groups[(currentIndex + 1) % groups.length]?.id ?? null;
}

/**
 * 判断左右键是否应交给输入控件原生光标导航。
 */
function shouldUseNativeHorizontalNavigation(event: KeyboardEvent) {
  const target = event.target;
  if (!(target instanceof HTMLElement)) return false;

  const tagName = target.tagName.toLowerCase();
  if (target.isContentEditable) return true;

  return tagName === "input" || tagName === "textarea";
}

/**
 * 当前选中分组被删除或不再存在时，回到全部分组。
 */
function ensureSelectedGroupStillExists(groups: ClipboardGroupRecord[]) {
  const selectedGroupId = clipboardViewState.groupId;
  if (!selectedGroupId) return;

  const exists = groups.some((record) => {
    return record.id === selectedGroupId;
  });
  if (exists) return;

  clipboardViewState.groupId = null;
}

export default Group;
