import Scrollbar from "@/components/Scrollbar";
import { useAutoAnimate } from "@formkit/auto-animate/react";
import { FloatButton } from "antd";
import { FixedSizeList } from "react-window";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);

	const outerRef = useRef(null);
	const virtualListRef = useRef<FixedSizeList>(null);
	const [animationParent, enableAnimations] = useAutoAnimate();

	useUpdateEffect(() => {
		virtualListRef.current?.scrollTo(0);

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
			<FixedSizeList
				ref={virtualListRef}
				outerRef={outerRef}
				innerRef={animationParent}
				width={360}
				height={506}
				itemData={state.historyList}
				itemKey={(index, data) => data[index].id}
				itemCount={state.historyList.length}
				itemSize={120}
				outerElementType={Scrollbar}
				onScroll={handleScroll}
				onItemsRendered={({ visibleStartIndex }) => {
					state.visibleStartIndex = visibleStartIndex;
				}}
			>
				{Item}
			</FixedSizeList>

			{/* @ts-ignore */}
			<FloatButton.BackTop duration={0} target={() => outerRef.current} />
		</>
	);
};

export default List;
