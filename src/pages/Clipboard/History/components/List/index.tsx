import Scrollbar from "@/components/Scrollbar";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);

	const containerTarget = useRef(null);
	const wrapperTarget = useRef(null);

	useUpdateEffect(() => {
		scrollTo(0);

		clipboardStore.activeIndex = 0;
	}, [state.group, state.isCollected]);

	const [list, scrollTo] = useVirtualList(state.historyList, {
		containerTarget,
		wrapperTarget,
		itemHeight: 120,
	});

	return (
		<Scrollbar ref={containerTarget} className="h-510">
			<div ref={wrapperTarget}>
				{list.map((props) => (
					<Item key={props.data.id} {...props} />
				))}
			</div>
		</Scrollbar>
	);
};

export default List;
