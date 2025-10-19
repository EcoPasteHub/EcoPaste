import Scrollbar from "@/components/Scrollbar";
import { FloatButton, Modal } from "antd";
import { findIndex } from "es-toolkit/compat";
import { type EventBusPayload, MainContext } from "../..";
import Item from "./components/Item";
import NoteModal, { type NoteModalRef } from "./components/NoteModal";

const HistoryList = () => {
	const { rootState } = useContext(MainContext);
	const noteModelRef = useRef<NoteModalRef>(null);
	const [deleteModal, contextHolder] = Modal.useModal();
	const scrollRef = useRef<HTMLDivElement>(null);
	const isFocusWithin = useFocusWithin(document.body);

	const { list, height, measureElement, scrollToIndex } =
		useHistoryList(scrollRef);

	const scrollToTop = () => {
		scrollToIndex(0);

		rootState.activeId = rootState.list[0].id;
	};

	useTauriListen(LISTEN_KEY.ACTIVATE_BACK_TOP, scrollToTop);

	useUpdateEffect(() => {
		if (rootState.list.length === 0) {
			rootState.activeId = void 0;
		}

		rootState.activeId ??= rootState.list[0]?.id;
	}, [rootState.list.length]);

	useEffect(() => {
		const index = findIndex(rootState.list, { id: rootState.activeId });

		if (index < 0) return;

		scrollToIndex(index);
	}, [rootState.activeId]);

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
		(event, key) => {
			if (key === "home") {
				return scrollToTop();
			}

			const { activeId } = rootState;

			if (!activeId) return;

			const payload: EventBusPayload = {
				id: activeId,
				action: "",
			};

			event?.preventDefault();

			switch (key) {
				// 空格预览
				case "space":
					payload.action = LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW;
					break;
				// 回车粘贴
				case "enter":
					payload.action = LISTEN_KEY.CLIPBOARD_ITEM_PASTE;
					break;
				// 删除
				case "backspace":
				case "delete":
					payload.action = LISTEN_KEY.CLIPBOARD_ITEM_DELETE;
					break;
				// 选中上一个
				case "uparrow":
					payload.action = LISTEN_KEY.CLIPBOARD_ITEM_SELECT_PREV;
					break;
				// 选中下一个
				case "downarrow":
					payload.action = LISTEN_KEY.CLIPBOARD_ITEM_SELECT_NEXT;
					break;
				// 收藏和取消收藏
				case PRESET_SHORTCUT.FAVORITE:
					payload.action = LISTEN_KEY.CLIPBOARD_ITEM_FAVORITE;
					break;
			}

			rootState.eventBus?.emit(payload);
		},
		{
			events: isFocusWithin ? [] : ["keydown"],
		},
	);

	return (
		<>
			<Scrollbar ref={scrollRef} offset={3} className="flex-1">
				<div data-tauri-drag-region className="relative" style={{ height }}>
					{list.map((item) => {
						const { key, start, index } = item;

						const data = rootState.list[index];

						if (!data) return;

						return (
							<div
								ref={measureElement}
								key={key}
								data-index={index}
								className="absolute w-full"
								style={{ top: start }}
							>
								<Item
									index={index}
									data={data}
									deleteModal={deleteModal}
									handleNote={() => noteModelRef.current?.open()}
								/>
							</div>
						);
					})}
				</div>
			</Scrollbar>

			<NoteModal ref={noteModelRef} />

			<FloatButton.BackTop
				duration={0}
				target={() => scrollRef.current!}
				onClick={scrollToTop}
			/>

			{contextHolder}
		</>
	);
};

export default HistoryList;
