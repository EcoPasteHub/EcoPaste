import Scrollbar from "@/components/Scrollbar";
import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import { useVirtualizer } from "@tanstack/react-virtual";
import { FloatButton } from "antd";
import { findIndex } from "lodash-es";
import { useSnapshot } from "valtio";
import Item from "./components/Item";

const List = () => {
	const { state, getClipboardList } = useContext(ClipboardPanelContext);
	const { env } = useSnapshot(globalStore);
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

	useEffect(() => {
		rowVirtualizer.scrollToIndex(0);

		getClipboardList?.();
	}, [state.search, state.group, state.isCollected]);

	// 空格预览
	useOSKeyPress("space", () => {
		state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW);
	});

	// 回车粘贴
	useOSKeyPress("enter", () => {
		state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_PASTE);
	});

	// 删除
	useOSKeyPress("backspace", () => {
		state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_DELETE);
	});

	// 选中上一个或者下一个
	useOSKeyPress(["uparrow", "downarrow"], (_, key) => {
		const index = findIndex(state.data.list, { id: state.activeId });

		let nextIndex = index;

		if (key === "uparrow") {
			if (index === 0) return;

			nextIndex = index - 1;
		} else {
			if (index === state.data.list.length - 1) return;

			nextIndex = index + 1;
		}

		state.activeId = state.data.list[nextIndex].id;

		state.scrollToIndex?.(nextIndex);
	});

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

						value = type !== "image" ? value : env.saveImageDir + value;

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
