import type { TreeDataNode } from "antd";
import { Button, Modal } from "antd";
import type { TFunction } from "i18next";
import type { FC, Key, MouseEvent } from "react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  createClipboardGroup,
  deleteClipboardGroup,
  updateClipboardGroup,
  updateClipboardGroupsLayout,
} from "@/commands";
import ClipboardGroupIcon from "@/components/ClipboardGroupIcon";
import ClipboardGroupModal from "@/components/ClipboardGroupModal";
import Dropdown, { type DropdownMenuItems } from "@/components/Dropdown";
import type {
  ClipboardGroupInput,
  ClipboardGroupRecord,
} from "@/types/clipboard";
import SortableTreeModal from "./settingControls/SortableTreeModal";

type GroupModalMode = "create" | "edit";
type GroupTreeAction = "delete" | "edit";

interface ClipboardGroupManagerModalProps {
  groups: ClipboardGroupRecord[];
  open: boolean;
  onCancel: () => void;
  onSaved?: () => void;
}

interface ClipboardGroupTreeTitleProps {
  record: ClipboardGroupRecord;
  onDelete: (record: ClipboardGroupRecord) => void;
  onEdit: (record: ClipboardGroupRecord) => void;
}

const GROUP_TREE_ACTION = {
  DELETE: "delete",
  EDIT: "edit",
} as const satisfies Record<string, GroupTreeAction>;

/**
 * 偏好设置里的自定义分组管理弹框，复用排序 Tree 与分组新增 / 编辑弹框。
 */
const ClipboardGroupManagerModal: FC<ClipboardGroupManagerModalProps> = (
  props,
) => {
  const { groups: initialGroups, open, onCancel, onSaved } = props;
  const { t } = useTranslation(["clipboard", "common"]);
  const [groups, setGroups] = useState<ClipboardGroupRecord[]>(initialGroups);
  const [treeOrder, setTreeOrder] = useState<string[]>([]);
  const [checkedKeys, setCheckedKeys] = useState<Key[]>([]);
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const [groupModalMode, setGroupModalMode] =
    useState<GroupModalMode>("create");
  const [treeResetKey, setTreeResetKey] = useState(0);
  const [editingGroup, setEditingGroup] = useState<ClipboardGroupRecord | null>(
    null,
  );
  const deleteGroupRef = useRef<ClipboardGroupRecord | null>(null);

  useEffect(() => {
    if (!open) return;

    setGroups(initialGroups);
    setTreeOrder(resolveInitialOrder(initialGroups));
    setCheckedKeys(resolveVisibleGroupIds(initialGroups));
    setTreeResetKey((current) => {
      return current + 1;
    });
  }, [initialGroups, open]);

  const openCreateModal = () => {
    setGroupModalMode("create");
    setEditingGroup(null);
    setGroupModalOpen(true);
  };

  const openEditModal = (record: ClipboardGroupRecord) => {
    setGroupModalMode("edit");
    setEditingGroup(record);
    setGroupModalOpen(true);
  };

  const closeGroupModal = () => {
    setGroupModalOpen(false);
    setEditingGroup(null);
  };

  const handleGroupSubmit = async (input: ClipboardGroupInput) => {
    if (groupModalMode === "create") {
      const record = await createClipboardGroup(input);

      setGroups((current) => {
        return [...current, record];
      });
      setTreeOrder((current) => {
        return [...current, record.id];
      });
      setCheckedKeys((current) => {
        if (record.isHidden) return current;

        return [...current, record.id];
      });
      resetTreeData();
      closeGroupModal();
      return;
    }

    if (!editingGroup) return;

    await updateClipboardGroup(editingGroup.id, input);
    setGroups((current) => {
      return current.map((record) => {
        if (record.id !== editingGroup.id) return record;

        return { ...record, ...input };
      });
    });
    resetTreeData();
    closeGroupModal();
  };

  const requestDeleteGroup = (record: ClipboardGroupRecord) => {
    deleteGroupRef.current = record;

    Modal.confirm({
      centered: true,
      content: (
        <span className="text-ant-secondary text-sm">
          {t("clipboard:groups.deleteConfirmDescription", {
            group: record.name,
          })}
        </span>
      ),
      okButtonProps: { danger: true },
      okText: t("common:actions.delete"),
      onOk: confirmDeleteGroup,
      title: t("clipboard:groups.delete"),
    });
  };

  const confirmDeleteGroup = async () => {
    const record = deleteGroupRef.current;
    if (!record) return;

    await deleteClipboardGroup(record.id);

    setGroups((current) => {
      return current.filter((item) => {
        return item.id !== record.id;
      });
    });
    setTreeOrder((current) => {
      return current.filter((id) => {
        return id !== record.id;
      });
    });
    setCheckedKeys((current) => {
      return current.filter((key) => {
        return key !== record.id;
      });
    });
    resetTreeData();
    deleteGroupRef.current = null;
  };

  const handleTreeDataChange = (nextTreeData: TreeDataNode[]) => {
    setTreeOrder(
      nextTreeData.map((node) => {
        return String(node.key);
      }),
    );
  };

  const handleCheckedKeysChange = (nextCheckedKeys: Key[]) => {
    setCheckedKeys(nextCheckedKeys);
  };

  const handleSave = async (nextOrder: string[], nextCheckedKeys: string[]) => {
    await updateClipboardGroupsLayout(nextOrder, nextCheckedKeys);
    onSaved?.();
  };

  const resetTreeData = () => {
    setTreeResetKey((current) => {
      return current + 1;
    });
  };

  const footerExtra = (
    <Button onClick={openCreateModal}>{t("common:actions.add")}</Button>
  );
  const treeData = buildTreeData(
    groups,
    treeOrder,
    openEditModal,
    requestDeleteGroup,
  );

  return (
    <>
      <SortableTreeModal
        cancelText={t("common:actions.cancel")}
        checkable
        checkedKeys={checkedKeys}
        footerExtra={footerExtra}
        okText={t("common:actions.save")}
        onCancel={onCancel}
        onCheckedKeysChange={handleCheckedKeysChange}
        onSave={handleSave}
        onTreeDataChange={handleTreeDataChange}
        open={open}
        resetKey={treeResetKey}
        title={t("clipboard:groups.manage")}
        treeData={treeData}
      />

      <ClipboardGroupModal
        group={editingGroup}
        mode={groupModalMode}
        onCancel={closeGroupModal}
        onSubmit={handleGroupSubmit}
        open={groupModalOpen}
      />
    </>
  );
};

export default ClipboardGroupManagerModal;

/**
 * 分组管理 Tree 的行内容：左侧图标和名称，右侧提供编辑 / 删除快捷动作。
 */
const ClipboardGroupTreeTitle: FC<ClipboardGroupTreeTitleProps> = (props) => {
  const { record, onDelete, onEdit } = props;
  const { t } = useTranslation(["clipboard"]);
  const menuItems = buildGroupTreeMenuItems(t);

  const stopTreeInteraction = (
    event: MouseEvent<HTMLButtonElement | HTMLSpanElement>,
  ) => {
    event.stopPropagation();
  };

  const handleMenuClick = (info: {
    key: string;
    domEvent?: { stopPropagation: () => void };
  }) => {
    info.domEvent?.stopPropagation();

    const action = parseGroupTreeAction(info.key);
    if (!action) return;

    if (action === GROUP_TREE_ACTION.EDIT) {
      onEdit(record);
      return;
    }

    if (action === GROUP_TREE_ACTION.DELETE) {
      onDelete(record);
    }
  };

  return (
    <span className="flex min-w-0 items-center justify-between gap-2">
      <span className="flex min-w-0 items-center gap-2">
        <ClipboardGroupIcon icon={record.icon} />
        <span className="min-w-0 truncate">{record.name}</span>
      </span>

      <span className="flex shrink-0 items-center gap-1">
        <Dropdown
          menu={{
            items: menuItems,
            onClick: handleMenuClick,
          }}
          trigger={["click"]}
        >
          <Button
            className="size-6"
            icon={<i aria-hidden="true" className="i-lucide:more-horizontal" />}
            onClick={stopTreeInteraction}
            onMouseDown={stopTreeInteraction}
            size="small"
            title={t("clipboard:groups.manage")}
            type="text"
          />
        </Dropdown>
      </span>
    </span>
  );
};

/**
 * 初始化 Tree 顺序，优先沿用数据库排序。
 */
function resolveInitialOrder(groups: ClipboardGroupRecord[]) {
  return groups.map((record) => {
    return record.id;
  });
}

/**
 * 初始化 Tree 勾选状态：勾选表示显示在主窗口分组栏。
 */
function resolveVisibleGroupIds(groups: ClipboardGroupRecord[]) {
  return groups
    .filter((record) => {
      return !record.isHidden;
    })
    .map((record) => {
      return record.id;
    });
}

/**
 * 根据当前排序生成单列 Tree 数据，缺失排序的新分组自动补到末尾。
 */
function buildTreeData(
  groups: ClipboardGroupRecord[],
  order: string[],
  onEdit: (record: ClipboardGroupRecord) => void,
  onDelete: (record: ClipboardGroupRecord) => void,
) {
  const groupById = new Map(
    groups.map((record) => {
      return [record.id, record];
    }),
  );
  const orderedIdSet = new Set(order);
  const orderedGroups = [
    ...order
      .map((id) => {
        return groupById.get(id);
      })
      .filter((record) => {
        return isClipboardGroupRecord(record);
      }),
    ...groups.filter((record) => {
      return !orderedIdSet.has(record.id);
    }),
  ];

  return orderedGroups.map((record) => {
    return {
      key: record.id,
      title: (
        <ClipboardGroupTreeTitle
          onDelete={onDelete}
          onEdit={onEdit}
          record={record}
        />
      ),
    };
  });
}

/**
 * 构建分组 Tree 行内更多菜单。
 */
function buildGroupTreeMenuItems(
  t: TFunction<["clipboard"]>,
): DropdownMenuItems {
  return [
    {
      icon: "i-lucide:pencil",
      key: GROUP_TREE_ACTION.EDIT,
      label: t("clipboard:groups.edit"),
    },
    {
      danger: true,
      icon: "i-lucide:trash-2",
      key: GROUP_TREE_ACTION.DELETE,
      label: t("clipboard:groups.delete"),
    },
  ];
}

/**
 * 解析分组 Tree 行内菜单动作。
 */
function parseGroupTreeAction(key: string): GroupTreeAction | null {
  const actions = Object.values(GROUP_TREE_ACTION);
  if (!actions.includes(key as GroupTreeAction)) return null;

  return key as GroupTreeAction;
}

/**
 * 收窄根据 id 查找分组后的可选值。
 */
function isClipboardGroupRecord(
  record: ClipboardGroupRecord | undefined,
): record is ClipboardGroupRecord {
  return record !== void 0;
}
