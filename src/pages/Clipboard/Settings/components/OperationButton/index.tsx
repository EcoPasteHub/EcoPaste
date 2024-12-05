import Icon from "@/components/Icon";
import ProListItem from "@/components/ProListItem";
import type { OperationButton as Key } from "@/types/store";
import { Button, Flex, Modal, Transfer, Tree, type TreeProps } from "antd";
import type { TransferCustomListBodyProps } from "antd/lib/transfer/list";
import { useSnapshot } from "valtio";

interface TransferData {
	key: Key;
	title: string;
	icon: string;
	activeIcon?: string;
}

export const transferData: TransferData[] = [
	{
		key: "copy",
		title:
			"preference.clipboard.content_settings.label.operation_button_option.copy",
		icon: "i-lucide:copy",
	},
	{
		key: "pastePlain",
		title:
			"preference.clipboard.content_settings.label.operation_button_option.paste_plain",
		icon: "i-lucide:clipboard-paste",
	},
	{
		key: "note",
		title:
			"preference.clipboard.content_settings.label.operation_button_option.notes",
		icon: "i-lucide:clipboard-pen-line",
	},
	{
		key: "star",
		title:
			"preference.clipboard.content_settings.label.operation_button_option.favorite",
		icon: "i-iconamoon:star",
		activeIcon: "i-iconamoon:star-fill",
	},
	{
		key: "delete",
		title:
			"preference.clipboard.content_settings.label.operation_button_option.delete",
		icon: "i-lucide:trash",
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
			<Flex key={key} align="center" gap={4} className="max-w-125">
				<Icon name={icon} />
				<span className="truncate">{t(title)}</span>
			</Flex>
		);
	};

	const renderTree = (data: TransferCustomListBodyProps<TransferData>) => {
		const { direction, selectedKeys, onItemSelect } = data;

		if (direction === "right" && content.operationButtons?.length) {
			return (
				<Tree
					checkable
					draggable
					blockNode
					className="[&_.ant-tree-switcher]:hidden"
					selectable={false}
					checkedKeys={selectedKeys}
					treeData={treeData}
					titleRender={renderTransferData}
					onCheck={(_, info) => {
						const { key } = info.node;

						onItemSelect(key, !selectedKeys?.includes(key));
					}}
					onDrop={handleDrop}
				/>
			);
		}
	};

	return (
		<>
			<ProListItem
				title={t(
					"preference.clipboard.content_settings.label.operation_button",
				)}
				description={t(
					"preference.clipboard.content_settings.hints.operation_button",
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
				open={open}
				title={t(
					"preference.clipboard.content_settings.label.custom_operation_button_title",
				)}
				width={448}
				footer={null}
				onCancel={toggle}
			>
				<Transfer
					dataSource={transferData}
					targetKeys={content.operationButtons}
					render={renderTransferData}
					onChange={(keys) => {
						clipboardStore.content.operationButtons = keys as Key[];
					}}
				>
					{renderTree}
				</Transfer>
			</Modal>
		</>
	);
};

export default OperationButton;
