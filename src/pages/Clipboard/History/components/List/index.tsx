import Scrollbar from "@/components/Scrollbar";
import { FloatButton } from "antd";
import { FixedSizeList } from "react-window";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);

	const outerRef = useRef(null);
	const virtualListRef = useRef<FixedSizeList>(null);

	useUpdateEffect(() => {
		virtualListRef.current?.scrollTo(0);

		state.activeIndex = 0;
	}, [state.search, state.group, state.isCollected]);

	return (
		<>
			<FixedSizeList
				ref={virtualListRef}
				outerRef={outerRef}
				width={360}
				height={506}
				itemData={state.historyList}
				itemKey={(index, data) => data[index].id}
				itemCount={state.historyList.length}
				itemSize={120}
				outerElementType={Scrollbar}
				onItemsRendered={({ visibleStartIndex }) => {
					state.visibleStartIndex = visibleStartIndex;
				}}
			>
				{Item}
			</FixedSizeList>

			{/* @ts-ignore */}
			<FloatButton.BackTop target={() => outerRef.current} />
		</>
	);
};

export default List;
