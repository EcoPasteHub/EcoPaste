import { FixedSizeList } from "react-window";
import { HistoryContext } from "../..";
import Item from "./components/Item";
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

export default List;
