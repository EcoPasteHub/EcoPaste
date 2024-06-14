import Icon from "@/components/Icon";
import type { HistoryItem } from "@/types/database";
import { Flex, Popconfirm } from "antd";
import clsx from "clsx";
import { FixedSizeList } from "react-window";
import {
	writeFilesURIs,
	writeHtml,
	writeImageBase64,
	writeRtf,
	writeText,
} from "tauri-plugin-clipboard-api";
import { HistoryContext } from "../..";
import Files from "./components/Files";
import Html from "./components/Html";
import Image from "./components/Image";
import Rtf from "./components/Rtf";
import Text from "./components/Text";

const Popup = () => {
	const { state, getHistoryList } = useContext(HistoryContext);

	const [visibleStartIndex, setVisibleStartIndex] = useState(0);

	const getChineseType = (data: HistoryItem) => {
		const { type, content = "" } = data;

		switch (type) {
			case "text":
				if (isURL(content)) {
					return "链接";
				}

				if (isEmail(content)) {
					return "邮箱";
				}

				return "文本";
			case "rtf":
				return "富文本";
			case "html":
				return "HTML";
			case "image":
				return "图片";
			case "files":
				return `${JSON.parse(content).length}个文件（夹）`;
		}
	};

	const writeContent = (data: HistoryItem) => {
		const { type, content = "" } = data;

		switch (type) {
			case "text":
				return writeText(content);
			case "rtf":
				return writeRtf(content);
			case "html":
				return writeHtml(content);
			case "image":
				return writeImageBase64(content);
			case "files":
				return writeFilesURIs(JSON.parse(content));
		}
	};

	const collect = async (data: HistoryItem) => {
		const { id, isCollected } = data;

		await updateSQL("history", {
			id,
			isCollected: !isCollected,
		});

		getHistoryList?.();
	};

	const renderContent = (data: HistoryItem) => {
		switch (data.type) {
			case "rtf":
				return <Rtf {...data} />;
			case "html":
				return <Html {...data} />;
			case "image":
				return <Image {...data} />;
			case "files":
				return <Files {...data} />;
			default:
				return <Text {...data} />;
		}
	};

	const deleteItem = async (id: number) => {
		await deleteSQL("history", id);

		getHistoryList?.();
	};

	return (
		<FixedSizeList
			width={336}
			height={542}
			itemData={state.historyList}
			itemKey={(index, data) => data[index].id!}
			itemCount={state.historyList.length}
			itemSize={120}
			onItemsRendered={({ visibleStartIndex }) =>
				setVisibleStartIndex(visibleStartIndex)
			}
		>
			{(item) => {
				const { index, style, data } = item;
				const historyData = data[index];
				const { id, createTime, isCollected } = historyData;

				const top = Number(style.top) + (index - visibleStartIndex) * 12;

				return (
					<Flex
						vertical
						gap={6}
						className="b b-color-2 hover:b-primary h-full rounded-6 bg-1 p-6 transition"
						style={{ ...style, top }}
						onDoubleClick={() => writeContent(historyData)}
					>
						<Flex justify="space-between" className="color-2">
							<Flex align="center" gap={6} className="text-12">
								<span>{getChineseType(historyData)}</span>
								<span>{dayjs(createTime).fromNow()}</span>
							</Flex>

							<Flex align="center" gap={6} className="text-14">
								<Icon
									hoverable
									name="i-iconamoon:copy"
									onMouseDown={() => writeContent(historyData)}
								/>
								<Icon
									hoverable
									name={
										isCollected ? "i-iconamoon:star-fill" : "i-iconamoon:star"
									}
									className={clsx({ "text-gold!": isCollected })}
									onMouseDown={() => collect(historyData)}
								/>
								<Popconfirm
									title="确定要删除吗？"
									placement="left"
									onConfirm={() => deleteItem(id!)}
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

						<div className="flex-1 overflow-hidden">
							{renderContent(historyData)}
						</div>
					</Flex>
				);
			}}
		</FixedSizeList>
	);
};

export default Popup;
