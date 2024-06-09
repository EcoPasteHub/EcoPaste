import Icon from "@/components/Icon";
import type { HistoryItem, HistoryType } from "@/types/database";
import { Flex } from "antd";
import clsx from "clsx";
import { FixedSizeList } from "react-window";
import { HistoryContext } from "../..";
import Files from "./components/Files";
import Html from "./components/Html";
import Image from "./components/Image";
import Rtf from "./components/Rtf";
import Text from "./components/Text";

const Popup = () => {
	const { state, getHistoryList } = useContext(HistoryContext);

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

	const getChineseType = (type: HistoryType) => {
		switch (type) {
			case "text":
				return "文本";
			case "rtf":
				return "富文本";
			case "html":
				return "HTML";
			case "image":
				return "图片";
			case "files":
				return "文件（夹）";
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

	return (
		<FixedSizeList
			width={336}
			height={542}
			itemData={state.historyList}
			itemKey={(index, data) => data[index].id!}
			itemCount={state.historyList.length}
			itemSize={120}
		>
			{(item) => {
				const { index, style, data } = item;
				const { type, createTime, isCollected } = data[index];

				return (
					<div
						data-tauri-drag-region
						style={style}
						className="not-last-of-type:pb-12"
					>
						<Flex
							vertical
							gap={6}
							className="h-full rounded-6 bg-white p-6 shadow"
						>
							<Flex justify="space-between" className="color-2 text-12">
								<span>{getChineseType(type!)}</span>
								<span>{createTime}</span>
								<Icon
									hoverable
									size={14}
									name={
										isCollected ? "i-iconamoon:star-fill" : "i-iconamoon:star"
									}
									className={clsx({
										"text-gold!": isCollected,
									})}
									onMouseDown={() => collect(data[index])}
								/>
							</Flex>

							<div className="flex-1 overflow-hidden">
								{renderContent(data[index])}
							</div>
						</Flex>
					</div>
				);
			}}
		</FixedSizeList>
	);
};

export default Popup;
