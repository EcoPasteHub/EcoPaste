import Icon from "@/components/Icon";
import type { HistoryItem } from "@/types/database";
import { Flex, Popconfirm } from "antd";
import clsx from "clsx";
import type { FC } from "react";
import {
	FixedSizeList,
	type ListChildComponentProps,
	areEqual,
} from "react-window";
import { HistoryContext } from "../..";
import Files from "./components/Item/components/Filescomponents/Files";
import HTML from "./components/Item/components/HTML/components/Files/HTML";
import Image from "./components/Item/components/Imagecomponents/Image";
import RichText from "./components/Item/components/RichTextponents/RichText";
import Text from "./components/Item/components/Text/components/Text";
import styles from "./index.module.scss";

const List = () => {
	const { state } = useContext(HistoryContext);

	const itemHeight = 132;
	const itemCount = state.historyList.length;

	return (
		<FixedSizeList
			width={336}
			height={542}
			itemData={state.historyList}
			itemKey={(index, data) => data[index].id!}
			itemCount={state.historyList.length}
			itemSize={itemHeight}
			className={styles.virtualList}
			style={{
				// @ts-ignore
				"--item-count": itemCount,
				"--item-height": `${itemHeight}px`,
			}}
		>
			{Item}
		</FixedSizeList>
	);
};

const Item: FC<ListChildComponentProps<HistoryItem[]>> = memo((props) => {
	const { getHistoryList } = useContext(HistoryContext);

	const { index, style, data } = props;

	const {
		id,
		type,
		value = "",
		size = 0,
		createTime,
		isCollected,
	} = data[index];

	const renderType = () => {
		switch (type) {
			case "text":
				if (isURL(value)) {
					return "链接";
				}

				if (isEmail(value)) {
					return "邮箱";
				}

				return "文本";
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
			return `${size}B`;
		}

		return `${size}个字符`;
	};

	const renderPixel = () => {
		const { width, height } = data[index];

		if (type !== "image") return;

		return (
			<span>
				{width}×{height}
			</span>
		);
	};

	const writeContent = () => {
		switch (type) {
			case "text":
				return writeText(value);
			case "rich-text":
				return writeRichText(value);
			case "html":
				return writeHTML(value);
			case "image":
				return writeImage(value);
			case "files":
				return writeFiles(JSON.parse(value));
		}
	};

	const collect = async () => {
		await updateSQL("history", { id, isCollected: !isCollected });

		getHistoryList?.();
	};

	const deleteItem = async () => {
		await deleteSQL("history", id);

		getHistoryList?.();
	};

	const renderContent = () => {
		const props = data[index];

		switch (type) {
			case "rich-text":
				return <RichText {...props} />;
			case "html":
				return <HTML {...props} />;
			case "image":
				return <Image {...props} />;
			case "files":
				return <Files {...props} />;
			default:
				return <Text {...props} />;
		}
	};

	return (
		<div style={style} className="last-of-type:h-120! not-last-of-type:pb-12">
			<Flex
				vertical
				gap={6}
				className="b b-color-2 hover:b-primary h-full rounded-6 bg-1 p-6 transition"
				onDoubleClick={writeContent}
			>
				<Flex justify="space-between" className="color-2">
					<Flex align="center" gap={6} className="text-12">
						<span>{renderType()}</span>
						<span>{renderSize()}</span>
						{renderPixel()}
						<span>{dayjs(createTime).fromNow()}</span>
					</Flex>

					<Flex align="center" gap={6} className="text-14">
						<Icon
							hoverable
							name="i-iconamoon:copy"
							onMouseDown={writeContent}
						/>
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

				<div className="flex-1 overflow-hidden">{renderContent()}</div>
			</Flex>
		</div>
	);
}, areEqual);

export default List;
