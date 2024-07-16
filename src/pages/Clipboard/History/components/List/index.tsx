import Scrollbar from "@/components/Scrollbar";
import { FloatButton } from "antd";
import { useSnapshot } from "valtio";
import { HistoryContext } from "../..";
import Item from "./components/Item";

const List = () => {
	const { state } = useContext(HistoryContext);
	const { saveImageDir } = useSnapshot(clipboardStore);

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
				{list.map((props) => {
					const { index, data } = props;
					const { type, value } = data;

					return (
						<Item
							key={props.data.id}
							index={index}
							data={{
								...data,
								value: type !== "image" ? value : saveImageDir + value,
							}}
						/>
					);
				})}
			</div>

			{/* @ts-ignore */}
			<FloatButton.BackTop target={() => containerTarget.current} />
		</Scrollbar>
	);
};

export default List;
