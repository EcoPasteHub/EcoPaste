import Scrollbar from "@/components/Scrollbar";
import { FloatButton } from "antd";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);

	const containerTarget = useRef(null);
	const wrapperTarget = useRef(null);

	const [list, scrollTo] = useVirtualList(state.historyList, {
		containerTarget,
		wrapperTarget,
		itemHeight: 120,
	});

	useMount(() => {
		state.scrollTo = scrollTo;
	});

	return (
		<Scrollbar ref={containerTarget} className="h-506">
			<div ref={wrapperTarget}>
				{list.map((props) => (
					<Item key={props.data.id} {...props} />
				))}
			</div>

			{/* @ts-ignore */}
			<FloatButton.BackTop target={() => containerTarget.current} />
		</Scrollbar>
	);
};

export default List;
