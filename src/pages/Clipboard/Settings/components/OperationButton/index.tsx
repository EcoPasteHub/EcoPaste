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
		title: "复制",
		icon: "i-lucide:copy",
	},
	{
		key: "pastePlain",
		title: "粘贴为纯文本",
		icon: "i-lucide:clipboard-paste",
	},
	{
		key: "note",
		title: "备注",
		icon: "i-lucide:clipboard-pen-line",
	},
	{
		key: "star",
		title: "收藏",
		icon: "i-iconamoon:star",
		activeIcon: "i-iconamoon:star-fill",
	},
	{
		key: "delete",
		title: "删除",
		icon: "i-lucide:trash",
	},
];

const OperationButton = () => {
	const { content } = useSnapshot(clipboardStore);
	const [open, { toggle }] = useBoolean();

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
			<Flex key={key} align="center" gap={4}>
				<Icon name={icon} />
				<span>{title}</span>
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
				title="操作按钮"
				description="自定义操作剪贴板内容的图标按钮"
			>
				<Button onClick={toggle}>自定义</Button>
			</ProListItem>

			<Modal
				centered
				open={open}
				title="自定义操作按钮"
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
