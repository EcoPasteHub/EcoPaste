import type { TreeDataNode, TreeProps } from "antd";
import { Button, Modal, Tree } from "antd";
import type { TFunction } from "i18next";
import type { FC, Key } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import Tooltip from "@/components/Tooltip";
import {
  isItemAction,
  resolveItemActionIcon,
  translateItemActionLabel,
} from "@/constants/itemActions";
import { cn } from "@/utils/cn";
import type {
  PreferenceOption,
  PreferenceSetting,
  SettingValue,
} from "../../types/preferences";
import {
  translatePreferenceControlLabel,
  translatePreferenceOption,
  translatePreferenceSetting,
} from "../../utils/preferenceI18n";
import type { ControlProps } from "./types";

interface SortableCheckboxTreeControlProps extends ControlProps {
  setting: PreferenceSetting;
  value?: SettingValue;
}

/**
 * 用一个按钮打开可拖拽 Tree 弹框，勾选项按当前树顺序保存。
 */
const SortableCheckboxTreeControl: FC<SortableCheckboxTreeControlProps> = (
  props,
) => {
  const { t } = useTranslation("preferences");
  const { t: clipboardT } = useTranslation("clipboard");
  const { t: commonT } = useTranslation("common");
  const { disabled, onChange, setting, value } = props;
  const treeValue = resolveTreeValue(value);
  const selectedLabels = resolveSelectedLabels(
    t,
    clipboardT,
    setting,
    treeValue.selected,
  );
  const controlLabel = translatePreferenceControlLabel(t, setting);
  const [open, setOpen] = useState(false);
  const [treeData, setTreeData] = useState<TreeDataNode[]>([]);
  const [checkedKeys, setCheckedKeys] = useState<Key[]>([]);

  if (setting.control.type !== "sortableCheckboxTree") return null;

  const openModal = () => {
    setTreeData(buildTreeData(t, clipboardT, setting, treeValue));
    setCheckedKeys(treeValue.selected);
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
  };

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

  const handleSave = async () => {
    const checkedSet = new Set(
      checkedKeys.map((key) => {
        return String(key);
      }),
    );
    const nextOrder = treeData.map((node) => {
      return String(node.key);
    });
    const nextSelected = nextOrder
      .map((key) => {
        return String(key);
      })
      .filter((key) => {
        return checkedSet.has(key);
      });

    await onChange(setting, { order: nextOrder, selected: nextSelected });
    setOpen(false);
  };

  return (
    <>
      <Tooltip title={selectedLabels}>
        <Button disabled={disabled} onClick={openModal}>
          {controlLabel}
        </Button>
      </Tooltip>

      <Modal
        cancelText={commonT("actions.cancel")}
        okText={commonT("actions.save")}
        onCancel={closeModal}
        onOk={handleSave}
        open={open}
        title={translatePreferenceSetting(t, setting, "title")}
      >
        <Tree
          allowDrop={allowTreeReorderDrop}
          blockNode
          checkable
          checkedKeys={checkedKeys}
          classNames={{
            item: "b b-ant-border-secondary rounded-md",
            itemSwitcher: "hidden",
          }}
          draggable
          onCheck={handleCheck}
          onDrop={handleDrop}
          selectable={false}
          treeData={treeData}
        />
      </Modal>
    </>
  );
};

export default SortableCheckboxTreeControl;

/**
 * 解析排序勾选树控件值；数组输入视为同时包含选择态和排序。
 */
function resolveTreeValue(value?: SettingValue) {
  if (Array.isArray(value)) {
    return { order: value, selected: value };
  }

  if (typeof value !== "object" || value === null) {
    return { order: [], selected: [] };
  }

  if (!("selected" in value) || !("order" in value)) {
    return { order: [], selected: [] };
  }

  const order = resolveStringArray(value.order);
  const selected = resolveStringArray(value.selected);

  return {
    order: order.length > 0 ? order : selected,
    selected,
  };
}

/**
 * 从未知值中提取字符串数组。
 */
function resolveStringArray(value: unknown) {
  if (!Array.isArray(value)) return [];

  return value.filter((item) => {
    return typeof item === "string";
  });
}

/**
 * 生成 Tooltip 里展示的已选动作摘要。
 */
function resolveSelectedLabels(
  t: TFunction<"preferences">,
  clipboardT: TFunction<"clipboard">,
  setting: PreferenceSetting,
  selectedValues: string[],
) {
  if (setting.control.type !== "sortableCheckboxTree") return "";

  const options = setting.control.options;
  const labels = selectedValues.map((value) => {
    const option = options.find((item) => {
      return String(item.value) === value;
    });
    if (!option) return value;

    return resolveOptionLabel(t, clipboardT, setting, value, option);
  });

  if (labels.length > 0) return labels.join(" / ");

  return translatePreferenceSetting(t, setting, "title");
}

/**
 * 根据保存的完整顺序与 schema 默认顺序生成单列 Tree 数据。
 */
function buildTreeData(
  t: TFunction<"preferences">,
  clipboardT: TFunction<"clipboard">,
  setting: PreferenceSetting,
  treeValue: { order: string[]; selected: string[] },
) {
  if (setting.control.type !== "sortableCheckboxTree") return [];

  const options = setting.control.options;
  const optionValues = options.map((option) => {
    return String(option.value);
  });
  const orderedSet = new Set(treeValue.order);
  const orderedValues = [
    ...treeValue.order,
    ...optionValues.filter((value) => {
      return !orderedSet.has(value);
    }),
  ];

  return orderedValues.reduce<TreeDataNode[]>((nodes, value) => {
    const option = options.find((item) => {
      return String(item.value) === value;
    });
    if (!option) return nodes;

    const label = resolveOptionLabel(t, clipboardT, setting, value, option);

    nodes.push({
      key: value,
      title: renderActionTitle(value, label),
    });

    return nodes;
  }, []);
}

/**
 * 渲染动作项标题；图标放在 checkbox 后、文案前。
 */
function renderActionTitle(value: string, label: string) {
  const iconClass = isItemAction(value)
    ? resolveItemActionIcon(value)
    : "i-lucide:circle";

  return (
    <span className="flex min-w-0 items-center gap-2">
      <i
        aria-hidden="true"
        className={cn(iconClass, "shrink-0 text-ant-secondary")}
      />
      <span className="min-w-0 truncate">{label}</span>
    </span>
  );
}

/**
 * 解析树节点选项文案；快捷动作复用剪贴板窗口同一组文案。
 */
function resolveOptionLabel(
  t: TFunction<"preferences">,
  clipboardT: TFunction<"clipboard">,
  setting: PreferenceSetting,
  value: string,
  option: PreferenceOption,
) {
  if (isItemAction(value)) return translateItemActionLabel(clipboardT, value);

  return String(translatePreferenceOption(t, setting, option).label);
}

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
