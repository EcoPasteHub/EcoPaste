import { FixedSizeList } from "react-window";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);

	return (
		<FixedSizeList
			width={336}
			height={542}
			itemData={state.historyList}
			itemKey={(index, data) => data[index].id!}
			itemCount={state.historyList.length}
			itemSize={120}
			onItemsRendered={({ visibleStartIndex }) => {
				clipboardStore.visibleStartIndex = visibleStartIndex;
			}}
		>
			{Item}
		</FixedSizeList>
	);
};

export default List;
