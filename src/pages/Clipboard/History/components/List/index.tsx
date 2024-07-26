import Scrollbar from "@/components/Scrollbar";
import { useVirtualizer } from "@tanstack/react-virtual";
import { FloatButton } from "antd";
import { useSnapshot } from "valtio";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state, getHistoryList } = useContext(HistoryContext);
	const { saveImageDir } = useSnapshot(clipboardStore);

	const outerRef = useRef<HTMLDivElement>(null);

	const rowVirtualizer = useVirtualizer({
		count: state.historyList.length,
		gap: 12,
		getScrollElement: () => outerRef.current,
		estimateSize: () => 120,
		getItemKey: (index) => state.historyList[index].id,
	});

	useEffect(() => {
		rowVirtualizer.scrollToIndex(0);

		getHistoryList?.();
	}, [state.search, state.group, state.isCollected]);

	return (
		<>
			<Scrollbar ref={outerRef} className="h-506">
				<div
					className="relative w-screen"
					style={{ height: rowVirtualizer.getTotalSize() }}
				>
					{rowVirtualizer.getVirtualItems().map((virtualItem) => {
						const { key, size, start, index } = virtualItem;
						const data = state.historyList[index];
						let { type, value } = data;

						value = type !== "image" ? value : saveImageDir + value;

						return (
							<Item
								key={key}
								index={index}
								data={{ ...data, value }}
								style={{ height: size, transform: `translateY(${start}px)` }}
							/>
						);
					})}
				</div>
			</Scrollbar>

			{/* @ts-ignore */}
			<FloatButton.BackTop duration={0} target={() => outerRef.current} />
		</>
	);
};

export default List;
