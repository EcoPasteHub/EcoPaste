import Scrollbar from "@/components/Scrollbar";
import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import { useVirtualizer } from "@tanstack/react-virtual";
import { FloatButton } from "antd";
import { findIndex } from "lodash-es";
import Item from "./components/Item";

const List = () => {
	const { state, getClipboardList } = useContext(ClipboardPanelContext);
	const outerRef = useRef<HTMLDivElement>(null);

	const rowVirtualizer = useVirtualizer({
		count: state.data.list.length,
		gap: 12,
		getScrollElement: () => outerRef.current,
		estimateSize: () => 120,
		getItemKey: (index) => state.data.list[index].id,
	});

	useMount(() => {
		state.scrollToIndex = rowVirtualizer.scrollToIndex;
	});

	const isFocusWithin = useFocusWithin(document.body);

	useEffect(() => {
		rowVirtualizer.scrollToIndex(0);

		getClipboardList?.();
	}, [state.search, state.group, state.isCollected]);

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
				const index = findIndex(state.data.list, { id: state.activeId });

				let nextIndex = index;

				if (key === "uparrow") {
					// 选中上一个
					if (index === 0) return;

					nextIndex = index - 1;
				} else {
					// 选中下一个
					if (index === state.data.list.length - 1) return;

					nextIndex = index + 1;
				}

				state.activeId = state.data.list[nextIndex].id;

				state.scrollToIndex?.(nextIndex);
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
						const data = state.data.list[index];
						let { type, value } = data;

						value = type !== "image" ? value : getSaveImageDir() + value;

						return (
							<Item
								key={key}
								index={index}
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
