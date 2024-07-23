import Scrollbar from "@/components/Scrollbar";
import { useAutoAnimate } from "@formkit/auto-animate/react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { FloatButton } from "antd";
import { useSnapshot } from "valtio";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);
	const { saveImageDir } = useSnapshot(clipboardStore);

	const outerRef = useRef<HTMLDivElement>(null);
	const [animationParent, enableAnimations] = useAutoAnimate();

	const rowVirtualizer = useVirtualizer({
		count: state.historyList.length,
		gap: 12,
		getScrollElement: () => outerRef.current,
		estimateSize: () => 120,
		getItemKey: (index) => state.historyList[index].id,
	});

	useUpdateEffect(() => {
		rowVirtualizer.scrollToIndex(0);

		state.activeIndex = 0;
	}, [state.search, state.group, state.isCollected]);

	const { run } = useDebounceFn(() => enableAnimations(true), {
		wait: 500,
	});

	const handleScroll = () => {
		enableAnimations(false);

		run();
	};

	return (
		<>
			<Scrollbar ref={outerRef} className="h-506" onScroll={handleScroll}>
				<div
					ref={animationParent}
					className="relative w-screen"
					style={{ height: rowVirtualizer.getTotalSize() }}
				>
					{rowVirtualizer.getVirtualItems().map((virtualItem) => {
						const { key, size, start, index } = virtualItem;
						const data = state.historyList[index];
						let { type, value } = data;

						value = type !== "image" ? value : saveImageDir + value;

						return (
							<div
								key={key}
								className="absolute inset-0"
								style={{
									height: size,
									transform: `translateY(${start}px)`,
								}}
							>
								<Item
									index={index}
									data={{
										...data,
										value,
									}}
								/>
							</div>
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
