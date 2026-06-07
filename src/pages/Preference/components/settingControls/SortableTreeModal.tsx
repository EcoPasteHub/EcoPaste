import type { TreeDataNode, TreeProps } from "antd";
import { Modal, Tree } from "antd";
import type { FC, Key, ReactNode } from "react";
import { useEffect, useRef, useState } from "react";

interface SortableTreeModalProps {
  cancelText: string;
  checkedKeys?: Key[];
  checkable?: boolean;
  okText: string;
  open: boolean;
  title: ReactNode;
  treeData: TreeDataNode[];
  onCancel: () => void;
  onSave: (order: string[], checkedKeys: string[]) => Promise<void>;
}

const EMPTY_CHECKED_KEYS: Key[] = [];

/**
 * 偏好设置里复用的可拖拽 Tree 弹框；支持纯排序，也支持排序 + 勾选。
 */
const SortableTreeModal: FC<SortableTreeModalProps> = (props) => {
  const {
    cancelText,
    checkedKeys: initialCheckedKeys = EMPTY_CHECKED_KEYS,
    checkable = false,
    okText,
    open,
    title,
    treeData: initialTreeData,
    onCancel,
    onSave,
  } = props;
  const [treeData, setTreeData] = useState<TreeDataNode[]>(initialTreeData);
  const [checkedKeys, setCheckedKeys] = useState<Key[]>(initialCheckedKeys);
  const [saving, setSaving] = useState(false);
  const openRef = useRef(open);

  useEffect(() => {
    const justOpened = open && !openRef.current;
    openRef.current = open;

    if (!justOpened) return;

    setTreeData(initialTreeData);
    setCheckedKeys(initialCheckedKeys);
  }, [open, initialTreeData, initialCheckedKeys]);

  const handleCheck: TreeProps["onCheck"] = (nextCheckedKeys) => {
    if (!Array.isArray(nextCheckedKeys)) {
      setCheckedKeys(nextCheckedKeys.checked);
      return;
    }

    setCheckedKeys(nextCheckedKeys);
  };

  const handleDrop: TreeProps["onDrop"] = (info) => {
    setTreeData((currentData) => {
      return reorderTreeData(currentData, info);
    });
  };

  const saveChanges = async () => {
    if (saving) return;

    const order = treeData.map((node) => {
      return String(node.key);
    });
    const selected = checkedKeys.map((key) => {
      return String(key);
    });

    setSaving(true);
    try {
      await onSave(order, selected);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      cancelText={cancelText}
      confirmLoading={saving}
      okText={okText}
      onCancel={onCancel}
      onOk={saveChanges}
      open={open}
      title={title}
    >
      <Tree
        allowDrop={allowTreeReorderDrop}
        blockNode
        checkable={checkable}
        checkedKeys={checkedKeys}
        classNames={{
          item: "py-1 b b-ant-border-secondary rounded-md",
          itemSwitcher: "hidden",
        }}
        draggable
        onCheck={handleCheck}
        onDrop={handleDrop}
        selectable={false}
        treeData={treeData}
      />
    </Modal>
  );
};

export default SortableTreeModal;

/**
 * 只允许拖到节点上方或下方，保持 Tree 为一列列表。
 */
const allowTreeReorderDrop: TreeProps["allowDrop"] = (info) => {
  return info.dropPosition !== 0;
};

/**
 * 按 Ant Design 官方 draggable Tree 示例的 dropPosition 逻辑调整单列顺序。
 */
function reorderTreeData(
  data: TreeDataNode[],
  info: Parameters<NonNullable<TreeProps["onDrop"]>>[0],
) {
  const nextData = [...data];
  const dropKey = String(info.node.key);
  const dragKey = String(info.dragNode.key);
  const dropPos = info.node.pos.split("-");
  const dropPosition = info.dropPosition - Number(dropPos[dropPos.length - 1]);
  const dragIndex = nextData.findIndex((node) => {
    return node.key === dragKey;
  });
  const dropIndex = nextData.findIndex((node) => {
    return node.key === dropKey;
  });

  if (dragIndex < 0 || dropIndex < 0) return nextData;

  const [dragNode] = nextData.splice(dragIndex, 1);
  const adjustedDropIndex = nextData.findIndex((node) => {
    return node.key === dropKey;
  });
  const insertIndex =
    dropPosition === -1 ? adjustedDropIndex : adjustedDropIndex + 1;

  nextData.splice(insertIndex, 0, dragNode);

  return nextData;
}
