import { open as openFileDialog } from "@tauri-apps/plugin-dialog";
import { Button, Form, type GetRef, Input, Modal } from "antd";
import type { FC, MouseEvent } from "react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { importClipboardGroupSvg, openExternalUrl } from "@/commands";
import { SVG_ICON_HELP_URL } from "@/constants/urls";
import type {
  ClipboardGroupIcon as ClipboardGroupIconValue,
  ClipboardGroupInput,
  ClipboardGroupRecord,
} from "@/types/clipboard";
import { cn } from "@/utils/cn";
import ClipboardGroupIcon from "../ClipboardGroupIcon";

const DEFAULT_GROUP_ICON = "i-lets-icons:folder";

const PRESET_GROUP_ICONS: ClipboardGroupIconValue[] = [
  DEFAULT_GROUP_ICON,
  "i-lets-icons:star",
  "i-lets-icons:book",
  "i-lets-icons:bookmark",
  "i-lets-icons:box",
  "i-lets-icons:database",
  "i-lets-icons:code",
  "i-lets-icons:link",
  "i-lets-icons:notebook",
  "i-lets-icons:calendar",
  "i-lets-icons:bell",
  "i-lets-icons:setting-line",
];

type GroupModalMode = "create" | "edit";
type InputRef = GetRef<typeof Input>;

interface ClipboardGroupFormValues {
  icon: ClipboardGroupIconValue;
  name: string;
}

interface ClipboardGroupModalProps {
  group: ClipboardGroupRecord | null;
  mode: GroupModalMode;
  onCancel: () => void;
  onSubmit: (input: ClipboardGroupInput) => Promise<void>;
  open: boolean;
}

/**
 * 判断图标值是否为自定义 SVG。
 */
const isCustomSvgIcon = (icon: ClipboardGroupIconValue) => {
  return icon.trimStart().startsWith("<svg");
};

/**
 * 生成分组弹框的初始表单值。
 */
const buildInitialValues = (
  group: ClipboardGroupRecord | null,
): ClipboardGroupFormValues => {
  return {
    icon: group?.icon ?? DEFAULT_GROUP_ICON,
    name: group?.name ?? "",
  };
};

/**
 * 自定义分组新增 / 编辑共享弹框。
 */
const ClipboardGroupModal: FC<ClipboardGroupModalProps> = (props) => {
  const { group, mode, onCancel, onSubmit, open } = props;
  const { t } = useTranslation(["clipboard", "common"]);
  const [form] = Form.useForm<ClipboardGroupFormValues>();

  const [submitting, setSubmitting] = useState(false);
  const [svgModalOpen, setSvgModalOpen] = useState(false);
  const nameInputRef = useRef<InputRef>(null);
  const icon = Form.useWatch("icon", form) ?? DEFAULT_GROUP_ICON;

  useEffect(() => {
    if (!open) return;

    form.setFieldsValue(buildInitialValues(group));
  }, [form, group, open]);

  /**
   * 弹框打开后聚焦名称输入框。
   */
  const handleAfterOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) return;

    requestAnimationFrame(() => {
      nameInputRef.current?.focus({ cursor: "end" });
    });
  };

  /**
   * 提交表单，交由调用方决定是新增还是更新。
   */
  const handleSubmit = async () => {
    const values = await form.validateFields();

    setSubmitting(true);

    try {
      await onSubmit({
        icon: values.icon,
        isHidden: group?.isHidden ?? false,
        name: values.name,
      });
    } finally {
      setSubmitting(false);
    }
  };

  /**
   * 选择一个预设图标。
   */
  const handlePresetIconClick = (event: MouseEvent<HTMLButtonElement>) => {
    const nextIcon = event.currentTarget.dataset.icon;
    if (!nextIcon) return;

    form.setFieldValue("icon", nextIcon);
  };

  /**
   * 打开自定义 SVG 管理弹框。
   */
  const openSvgModal = () => {
    setSvgModalOpen(true);
  };

  /**
   * 关闭自定义 SVG 管理弹框。
   */
  const closeSvgModal = () => {
    setSvgModalOpen(false);
  };

  /**
   * 使用 Tauri dialog 选择 SVG 文件，并交给 Rust 读取和校验。
   */
  const importSvg = async () => {
    const selected = await openFileDialog({
      filters: [{ extensions: ["svg"], name: "SVG" }],
      multiple: false,
    });
    if (typeof selected !== "string") return;

    const svg = await importClipboardGroupSvg(selected);
    form.setFieldValue("icon", svg);
  };

  /**
   * 删除当前自定义 SVG，回退到默认预设图标。
   */
  const removeCustomIcon = () => {
    form.setFieldValue("icon", DEFAULT_GROUP_ICON);
  };

  /**
   * 打开 SVG 图标下载说明链接。
   */
  const openIconHelp = async () => {
    await openExternalUrl(SVG_ICON_HELP_URL);
  };

  const title =
    mode === "create" ? t("clipboard:groups.add") : t("clipboard:groups.edit");
  const customIconSelected = isCustomSvgIcon(icon);

  return (
    <>
      <Modal
        afterOpenChange={handleAfterOpenChange}
        confirmLoading={submitting}
        destroyOnHidden
        mask={{ closable: false }}
        okText={t("common:actions.save")}
        onCancel={onCancel}
        onOk={handleSubmit}
        open={open}
        title={title}
      >
        <Form<ClipboardGroupFormValues>
          form={form}
          initialValues={buildInitialValues(group)}
          layout="vertical"
        >
          <Form.Item
            label={t("clipboard:groups.name")}
            name="name"
            rules={[{ required: true, whitespace: true }]}
          >
            <Input
              maxLength={32}
              placeholder={t("clipboard:groups.namePlaceholder")}
              ref={nameInputRef}
            />
          </Form.Item>

          <Form.Item hidden name="icon">
            <Input />
          </Form.Item>

          <Form.Item label={t("clipboard:groups.icon")}>
            <div className="flex flex-col gap-3">
              <div className="grid grid-cols-6 gap-2">
                {PRESET_GROUP_ICONS.map((presetIcon) => {
                  const selected = icon === presetIcon;

                  return (
                    <button
                      className={cn(
                        "flex size-9 cursor-pointer items-center justify-center rounded-2 border border-ant-border bg-ant-container transition-colors",
                        {
                          "border-ant-primary bg-ant-primary": selected,
                          "hover:bg-ant-fill-tertiary": !selected,
                        },
                      )}
                      data-icon={presetIcon}
                      key={presetIcon}
                      onClick={handlePresetIconClick}
                      type="button"
                    >
                      <ClipboardGroupIcon
                        icon={presetIcon}
                        selected={selected}
                        size="md"
                      />
                    </button>
                  );
                })}
              </div>

              <Button
                icon={
                  <ClipboardGroupIcon
                    icon={icon}
                    selected={customIconSelected}
                    size="md"
                  />
                }
                onClick={openSvgModal}
              >
                {t(
                  customIconSelected
                    ? "clipboard:groups.customIcon"
                    : "clipboard:groups.useCustomIcon",
                )}
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        destroyOnHidden
        footer={null}
        mask={{ closable: false }}
        onCancel={closeSvgModal}
        open={svgModalOpen}
        title={t("clipboard:groups.customIcon")}
      >
        <div className="flex flex-col gap-4">
          <div className="flex min-h-24 items-center justify-center rounded-2 border border-ant-border bg-ant-fill-quaternary">
            <ClipboardGroupIcon icon={icon} size="lg" />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button
              icon={<i className="i-lucide:upload text-sm!" />}
              onClick={importSvg}
            >
              {t("clipboard:groups.importSvg")}
            </Button>

            <Button
              disabled={!customIconSelected}
              icon={<i className="i-lucide:trash-2 text-sm!" />}
              onClick={removeCustomIcon}
            >
              {t("clipboard:groups.removeIcon")}
            </Button>
          </div>

          <Button
            icon={<i className="i-lucide:external-link text-sm!" />}
            onClick={openIconHelp}
            type="link"
          >
            {t("clipboard:groups.iconHelp")}
          </Button>
        </div>
      </Modal>
    </>
  );
};

export default ClipboardGroupModal;
