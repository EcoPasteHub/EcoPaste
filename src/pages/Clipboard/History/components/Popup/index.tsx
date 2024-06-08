import type { HistoryItem } from "@/types/database";
import { Flex } from "antd";
import { FixedSizeList } from "react-window";
import { HistoryContext } from "../..";
import Files from "./components/Files";
import Html from "./components/Html";
import Image from "./components/Image";
import Rtf from "./components/Rtf";
import Text from "./components/Text";

const Popup = () => {
	const { state } = useContext(HistoryContext);

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
				const { type, createTime } = data[index];

				return (
					<div
						data-tauri-drag-region
						style={style}
						className="not-last-of-type:pb-12"
					>
						<div className="h-full overflow-hidden rounded-6 bg-white p-6 shadow">
							<Flex justify="space-between" className="pb-6 text-12">
								<span>{type}</span>
								<span>{createTime}</span>
							</Flex>

							<div className="overflow-hidden">
								{renderContent(data[index])}
							</div>
						</div>
					</div>
				);
			}}
		</FixedSizeList>
	);
};

export default Popup;
