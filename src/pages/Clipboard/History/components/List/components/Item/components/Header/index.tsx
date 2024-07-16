import Icon from "@/components/Icon";
import type { HistoryItem } from "@/types/database";
import { Flex, Popconfirm } from "antd";
import { autoConvertBytes } from "arcdash";
import clsx from "clsx";
import type { FC } from "react";

interface HeaderProps extends HistoryItem {
	copy: () => void;
	collect: () => void;
	deleteItem: () => void;
}

const Header: FC<HeaderProps> = (props) => {
	const {
		type,
		value,
		size,
		createTime,
		isCollected,
		copy,
		collect,
		deleteItem,
	} = props;

	const renderType = () => {
		switch (type) {
			case "text":
				if (isURL(value)) {
					return "链接";
				}

				if (isEmail(value)) {
					return "邮箱";
				}

				if (isColor(value)) {
					return "颜色";
				}

				return "纯文本";
			case "rich-text":
				return "富文本";
			case "html":
				return "HTML";
			case "image":
				return "图片";
			case "files":
				return `${JSON.parse(value).length}个文件（夹）`;
		}
	};

	const renderSize = () => {
		if (type === "files" || type === "image") {
			return autoConvertBytes(size);
		}

		return `${size}个字符`;
	};

	const renderPixel = () => {
		if (type !== "image") return;

		const { width, height } = props;

		return (
			<span>
				{width}×{height}
			</span>
		);
	};

	return (
		<Flex justify="space-between" className="color-2">
			<Flex align="center" gap={6} className="text-12">
				<span>{renderType()}</span>
				<span>{renderSize()}</span>
				{renderPixel()}
				<span>{dayjs(createTime).fromNow()}</span>
			</Flex>

			<Flex align="center" gap={6} className="text-14">
				<Icon hoverable name="i-iconamoon:copy" onMouseDown={copy} />

				<Icon
					hoverable
					name={isCollected ? "i-iconamoon:star-fill" : "i-iconamoon:star"}
					className={clsx({ "text-gold!": isCollected })}
					onMouseDown={collect}
				/>

				<Popconfirm
					title="确定删除该历史记录？"
					placement="left"
					onConfirm={deleteItem}
				>
					<Icon
						hoverable
						size={15}
						name="i-iconamoon:trash-simple"
						className="hover:text-red!"
					/>
				</Popconfirm>
			</Flex>
		</Flex>
	);
};

export default memo(Header);
