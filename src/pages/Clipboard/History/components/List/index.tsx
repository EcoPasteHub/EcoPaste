import { MacScrollbar } from "mac-scrollbar";
import { FixedSizeList } from "react-window";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);

	const virtualListRef = useRef<FixedSizeList>(null);

	useUpdateEffect(() => {
		virtualListRef.current?.scrollTo(0);
		clipboardStore.activeIndex = 0;
	}, [state.group, state.isCollected]);

	return (
		<FixedSizeList
			ref={virtualListRef}
			width={360}
			height={542}
			itemData={state.historyList}
			itemKey={(index, data) => data[index].id!}
			itemCount={state.historyList.length}
			itemSize={120}
			outerElementType={MacScrollbar}
			onItemsRendered={({ visibleStartIndex }) => {
				clipboardStore.visibleStartIndex = visibleStartIndex;
			}}
		>
			{Item}
		</FixedSizeList>
	);
};

export default List;
