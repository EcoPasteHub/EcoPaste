import type { TreeDataNode } from "antd";
import { Button } from "antd";
import type { TFunction } from "i18next";
import type { FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import Tooltip from "@/components/Tooltip";
import {
  CAPTURE_KIND_ORDER,
  isCaptureKind,
  resolveCaptureKindIcon,
  translateCaptureKindLabel,
} from "@/constants/captureKinds";
import type { CaptureKind } from "@/types/settings";
import { cn } from "@/utils/cn";
import type { PreferenceSetting, SettingValue } from "../../types/preferences";
import {
  translatePreferenceControlLabel,
  translatePreferenceSetting,
} from "../../utils/preferenceI18n";
import SortableTreeModal from "./SortableTreeModal";
import type { ControlProps } from "./types";

interface CaptureOrderControlProps extends ControlProps {
  setting: PreferenceSetting;
  value?: SettingValue;
}

/**
 * 通过可拖拽 Tree 弹框调整剪贴板多表示内容的采集优先级。
 */
const CaptureOrderControl: FC<CaptureOrderControlProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { t: commonT } = useTranslation("common");
  const { disabled, onChange, setting, value } = props;
  const captureOrder = resolveCaptureOrder(value);
  const selectedLabels = resolveCaptureOrderLabels(t, captureOrder);
  const controlLabel = translatePreferenceControlLabel(t, setting);
  const [open, setOpen] = useState(false);
  const [treeData, setTreeData] = useState<TreeDataNode[]>([]);

  if (setting.control.type !== "sortableTree") return null;

  const openModal = () => {
    setTreeData(buildTreeData(t, setting, captureOrder));
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
  };

  const handleSave = async (nextOrder: string[]) => {
    await onChange(setting, resolveCaptureOrder(nextOrder));
    setOpen(false);
  };

  return (
    <>
      <Tooltip title={selectedLabels}>
        <Button disabled={disabled} onClick={openModal}>
          {controlLabel}
        </Button>
      </Tooltip>

      <SortableTreeModal
        cancelText={commonT("actions.cancel")}
        okText={commonT("actions.save")}
        onCancel={closeModal}
        onSave={handleSave}
        open={open}
        title={translatePreferenceSetting(t, setting, "title")}
        treeData={treeData}
      />
    </>
  );
};

export default CaptureOrderControl;

/**
 * 解析采集顺序设置值，并补齐缺失项、过滤未知项。
 */
function resolveCaptureOrder(value?: SettingValue): CaptureKind[] {
  const input = Array.isArray(value) ? value : [];
  const order: CaptureKind[] = [];

  for (const item of [...input, ...CAPTURE_KIND_ORDER]) {
    if (typeof item !== "string") {
      continue;
    }

    if (!isCaptureKind(item) || order.includes(item)) {
      continue;
    }

    order.push(item);
  }

  return order;
}

/**
 * 生成按钮 Tooltip 中展示的当前采集优先级摘要。
 */
function resolveCaptureOrderLabels(
  t: TFunction<"preferences">,
  order: CaptureKind[],
) {
  return order
    .map((kind) => {
      return translateCaptureKindLabel(t, kind);
    })
    .join(" / ");
}

/**
 * 根据当前顺序生成单列 Tree 节点。
 */
function buildTreeData(
  t: TFunction<"preferences">,
  setting: PreferenceSetting,
  order: CaptureKind[],
) {
  if (setting.control.type !== "sortableTree") return [];

  return order.reduce<TreeDataNode[]>((nodes, kind) => {
    const option = setting.control.options.find((item) => {
      return item.value === kind;
    });
    if (!option) return nodes;

    nodes.push({
      key: kind,
      title: renderCaptureTitle(kind, translateCaptureKindLabel(t, kind)),
    });

    return nodes;
  }, []);
}

/**
 * 渲染采集类型项标题；图标放在文案前，便于拖拽列表快速扫描。
 */
function renderCaptureTitle(kind: CaptureKind, label: string) {
  return (
    <span className="flex min-w-0 items-center gap-2">
      <i
        aria-hidden="true"
        className={cn(
          resolveCaptureKindIcon(kind),
          "shrink-0 text-ant-secondary",
        )}
      />
      <span className="min-w-0 truncate">{label}</span>
    </span>
  );
}
