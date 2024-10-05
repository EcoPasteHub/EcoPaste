import Scrollbar from "@/components/Scrollbar";
import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import { useVirtualizer } from "@tanstack/react-virtual";
import { FloatButton } from "antd";
import { findIndex } from "lodash-es";
import Item from "./components/Item";
import RemarkModal, { type RemarkModalRef } from "./components/RemarkModel";

const List = () => {
	const { state, getList } = useContext(ClipboardPanelContext);
	const outerRef = useRef<HTMLDivElement>(null);
	const remarkModelRef = useRef<RemarkModalRef>(null);

	const rowVirtualizer = useVirtualizer({
		count: state.list.length,
		gap: 12,
		getScrollElement: () => outerRef.current,
		estimateSize: () => 120,
		getItemKey: (index) => state.list[index].id,
	});

	useMount(() => {
		state.scrollToIndex = rowVirtualizer.scrollToIndex;
	});

	const isFocusWithin = useFocusWithin(document.body);

	useAsyncEffect(async () => {
		rowVirtualizer.scrollToIndex(0);

		await getList?.();

		state.activeId = state.list[0]?.id;
	}, [state.search, state.group, state.favorite]);

	// 滚动到选中
	useEffect(() => {
		const index = findIndex(state.list, { id: state.activeId });

		if (index < 0) return;

		rowVirtualizer.scrollToIndex?.(index);
	}, [state.activeId]);

	// 始终保持有一个选中
	useUpdateEffect(() => {
		if (state.list.length === 0) {
			state.activeId = undefined;
		}

		state.activeId ??= state.list[0]?.id;
	}, [state.list.length]);

	useOSKeyPress(
		["space", "enter", "backspace", "uparrow", "downarrow", "home"],
		(_, key) => {
			state.eventBusId = state.activeId;

			switch (key) {
				// 空格预览
				case "space":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW);
				// 回车粘贴
				case "enter":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_PASTE);
				// 删除
				case "backspace":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_DELETE);
				// 选中上一个
				case "uparrow":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_SELECT_PREV);
				// 选中下一个
				case "downarrow":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_SELECT_NEXT);
				// 回到顶部
				case "home":
					return rowVirtualizer.scrollToIndex?.(0);
			}
		},
		{
			events: isFocusWithin ? [] : ["keydown"],
		},
	);

	return (
		<>
			<Scrollbar ref={outerRef} className="flex-1">
				<div
					data-tauri-drag-region
					className="relative w-full"
					style={{ height: rowVirtualizer.getTotalSize() }}
				>
					{rowVirtualizer.getVirtualItems().map((virtualItem) => {
						const { key, size, start, index } = virtualItem;
						const data = state.list[index];
						let { type, value } = data;

						value = type !== "image" ? value : getSaveImagePath(value);

						return (
							<Item
								key={key}
								index={index}
								data={{ ...data, value }}
								style={{ height: size, transform: `translateY(${start}px)` }}
								openRemarkModel={() => remarkModelRef.current?.open()}
							/>
						);
					})}
				</div>
			</Scrollbar>

			{/* @ts-ignore */}
			<FloatButton.BackTop duration={0} target={() => outerRef.current} />

			<RemarkModal ref={remarkModelRef} />
		</>
	);
};

export default List;
