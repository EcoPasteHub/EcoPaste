import { Button, Flex, Modal, Transfer, Tree, type TreeProps } from "antd";
import type { TransferCustomListBodyProps } from "antd/lib/transfer/list";
import { useSnapshot } from "valtio";
import ProListItem from "@/components/ProListItem";
import UnoIcon from "@/components/UnoIcon";
import type { OperationButton as Key } from "@/types/store";

interface TransferData {
  key: Key;
  title: string;
  icon: string;
  activeIcon?: string;
}

export const transferData: TransferData[] = [
  {
    icon: "i-lucide:copy",
    key: "copy",
    title:
      "preference.clipboard.content_settings.label.operation_button_option.copy",
  },
  {
    icon: "i-lucide:clipboard-paste",
    key: "pastePlain",
    title:
      "preference.clipboard.content_settings.label.operation_button_option.paste_plain",
  },
  {
    icon: "i-lucide:clipboard-pen-line",
    key: "note",
    title:
      "preference.clipboard.content_settings.label.operation_button_option.notes",
  },
  {
    activeIcon: "i-iconamoon:star-fill",
    icon: "i-iconamoon:star",
    key: "star",
    title:
      "preference.clipboard.content_settings.label.operation_button_option.favorite",
  },
  {
    icon: "i-lucide:trash",
    key: "delete",
    title:
      "preference.clipboard.content_settings.label.operation_button_option.delete",
  },
];

const OperationButton = () => {
  const { content } = useSnapshot(clipboardStore);
  const [open, { toggle }] = useBoolean();
  const { t } = useTranslation();

  const treeData = useCreation(() => {
    return content.operationButtons.map((key) => {
      return transferData.find((data) => data.key === key)!;
    });
  }, [content.operationButtons]);

  const handleDrop: TreeProps["onDrop"] = (info) => {
    const { dragNode, node, dropPosition } = info;

    const getIndex = (pos: string) => pos.split("-").map(Number)[1];

    const dragIndex = getIndex(dragNode.pos);
    let dropIndex = getIndex(node.pos);

    if (dragIndex > dropIndex && dropPosition > 0) {
      dropIndex++;
    }

    const buttons = clipboardStore.content.operationButtons;
    buttons.splice(dropIndex, 0, ...buttons.splice(dragIndex, 1));
  };

  const renderTransferData = (data: TransferData) => {
    const { key, icon, title } = data;

    return (
      <Flex align="center" className="max-w-31.25" gap={4} key={key}>
        <UnoIcon name={icon} />
        <span className="truncate">{t(title)}</span>
      </Flex>
    );
  };

  const renderTree = (data: TransferCustomListBodyProps<TransferData>) => {
    const { direction, selectedKeys, onItemSelect } = data;

    if (direction === "right" && content.operationButtons?.length) {
      return (
        <Tree
          blockNode
          checkable
          checkedKeys={selectedKeys}
          className="[&_.ant-tree-switcher]:hidden"
          draggable
          onCheck={(_, info) => {
            const { key } = info.node;

            onItemSelect(key, !selectedKeys?.includes(key));
          }}
          onDrop={handleDrop}
          selectable={false}
          titleRender={renderTransferData}
          treeData={treeData}
        />
      );
    }
  };

  return (
    <>
      <ProListItem
        description={t(
          "preference.clipboard.content_settings.hints.operation_button",
        )}
        title={t(
          "preference.clipboard.content_settings.label.operation_button",
        )}
      >
        <Button onClick={toggle}>
          {t(
            "preference.clipboard.content_settings.button.custom_operation_button",
          )}
        </Button>
      </ProListItem>

      <Modal
        centered
        destroyOnClose
        footer={null}
        onCancel={toggle}
        open={open}
        title={t(
          "preference.clipboard.content_settings.label.custom_operation_button_title",
        )}
        width={448}
      >
        <Transfer
          dataSource={transferData}
          onChange={(keys) => {
            clipboardStore.content.operationButtons = keys as Key[];
          }}
          render={renderTransferData}
          targetKeys={content.operationButtons}
        >
          {renderTree}
        </Transfer>
      </Modal>
    </>
  );
};

export default OperationButton;
