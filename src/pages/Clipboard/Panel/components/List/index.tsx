import Scrollbar from "@/components/Scrollbar";
import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import { useVirtualizer } from "@tanstack/react-virtual";
import { FloatButton } from "antd";
import { findIndex } from "lodash-es";
import Item from "./components/Item";

const List = () => {
	const { state, getList } = useContext(ClipboardPanelContext);
	const outerRef = useRef<HTMLDivElement>(null);

	const rowVirtualizer = useVirtualizer({
		count: state.list.length,
		gap: 12,
		getScrollElement: () => outerRef.current,
		estimateSize: () => 120,
		getItemKey: (index) => state.list[index].id,
	});

	const isFocusWithin = useFocusWithin(document.body);

	useAsyncEffect(async () => {
		rowVirtualizer.scrollToIndex(0);

		await getList?.();

		state.activeId = state.list[0]?.id;
	}, [state.search, state.group, state.favorite]);

	useOSKeyPress(
		["space", "enter", "backspace", "uparrow", "downarrow"],
		(_, key) => {
			if (key === "space") {
				// 空格预览
				state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW);
			} else if (key === "enter") {
				// 回车粘贴
				state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_PASTE);
			} else if (key === "backspace") {
				// 删除
				state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_DELETE);
			} else {
				const index = findIndex(state.list, { id: state.activeId });

				let nextIndex = index;

				if (key === "uparrow") {
					// 选中上一个
					if (index === 0) return;

					nextIndex = index - 1;
				} else {
					// 选中下一个
					if (index === state.list.length - 1) return;

					nextIndex = index + 1;
				}

				state.activeId = state.list[nextIndex].id;

				rowVirtualizer.scrollToIndex?.(nextIndex);
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
								data={{ ...data, value }}
								style={{ height: size, transform: `translateY(${start}px)` }}
							/>
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
