import Scrollbar from "@/components/Scrollbar";
import { MainContext } from "@/pages/Main";
import { useVirtualizer } from "@tanstack/react-virtual";
import { FloatButton, Modal } from "antd";
import { findIndex } from "lodash-es";
import Item from "./components/Item";
import NoteModal, { type NoteModalRef } from "./components/NoteModal";

const List = () => {
	const { state, getList } = useContext(MainContext);
	const outerRef = useRef<HTMLDivElement>(null);
	const noteModelRef = useRef<NoteModalRef>(null);
	const [deleteModal, contextHolder] = Modal.useModal();

	const rowVirtualizer = useVirtualizer({
		count: state.list.length,
		gap: 8,
		getScrollElement: () => outerRef.current,
		estimateSize: () => 120,
		getItemKey: (index) => state.list[index].id,
	});

	// 监听激活时回到顶部并选中第一个
	useTauriListen(LISTEN_KEY.ACTIVATE_BACK_TOP, () => scrollToTop());

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
			state.activeId = void 0;
		}

		state.activeId ??= state.list[0]?.id;
	}, [state.list.length]);

	useKeyPress(
		[
			"space",
			"enter",
			"backspace",
			"delete",
			"uparrow",
			"downarrow",
			"home",
			PRESET_SHORTCUT.FAVORITE,
		],
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
				case "delete":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_DELETE);
				// 选中上一个
				case "uparrow":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_SELECT_PREV);
				// 选中下一个
				case "downarrow":
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_SELECT_NEXT);
				// 回到顶部
				case "home":
					return scrollToTop();
				// 收藏和取消收藏
				case PRESET_SHORTCUT.FAVORITE:
					return state.$eventBus?.emit(LISTEN_KEY.CLIPBOARD_ITEM_FAVORITE);
			}
		},
		{
			events: isFocusWithin ? [] : ["keydown"],
		},
	);

	// 回到顶部并选中第一个
	const scrollToTop = () => {
		rowVirtualizer.scrollToIndex(0);

		state.activeId = state.list[0]?.id;
	};

	return (
		<>
			<Scrollbar ref={outerRef} offset={3} className="flex-1">
				<div
					data-tauri-drag-region
					className="relative w-full"
					style={{ height: rowVirtualizer.getTotalSize() }}
				>
					{rowVirtualizer.getVirtualItems().map((virtualItem) => {
						const { key, size, start, index } = virtualItem;
						const data = state.list[index];
						let { type, value } = data;

						value = type !== "image" ? value : resolveImagePath(value);

						return (
							<Item
								key={key}
								index={index}
								data={{ ...data, value }}
								deleteModal={deleteModal}
								openNoteModel={() => noteModelRef.current?.open()}
								style={{ height: size, transform: `translateY(${start}px)` }}
							/>
						);
					})}
				</div>
			</Scrollbar>

			<FloatButton.BackTop
				duration={0}
				target={() => outerRef.current!}
				onClick={scrollToTop}
			/>

			<NoteModal ref={noteModelRef} />

			{contextHolder}
		</>
	);
};

export default List;
