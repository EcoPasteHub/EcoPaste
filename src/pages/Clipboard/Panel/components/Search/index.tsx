import Icon from "@/components/Icon";
import type { InputRef } from "antd";
import { Input } from "antd";
import type { FC, HTMLAttributes } from "react";
import { ClipboardPanelContext } from "../..";

const Search: FC<HTMLAttributes<HTMLDivElement>> = (props) => {
	const { state } = useContext(ClipboardPanelContext);
	const inputRef = useRef<InputRef>(null);
	const [value, setValue] = useState<string>();
	const [isComposition, { setTrue, setFalse }] = useBoolean();
	const { t } = useTranslation();

	useEffect(() => {
		if (isComposition) return;

		state.search = value;
	}, [value, isComposition]);

	useFocus({
		onFocus() {
			const { window, search } = clipboardStore;

			// 搜索框默认聚焦
			if (search.defaultFocus) {
				inputRef.current?.focus();
			} else {
				inputRef.current?.blur();
			}

			// 激活窗口时滚动到顶部并选中首项
			if (window.backTop) {
				requestAnimationFrame(() => {
					state.scrollToIndex?.(0);

					state.activeId = state.list[0]?.id;
				});
			}
		},
		onBlur() {
			const { search } = clipboardStore;

			// 搜索框自动清空
			if (search.autoClear) {
				setValue(undefined);
			}
		},
	});

	useOSKeyPress(["meta.f", "ctrl.f"], () => {
		inputRef.current?.focus();
	});

	useKeyPress(
		["enter", "uparrow", "downarrow"],
		() => {
			inputRef.current?.blur();
		},
		{
			target: inputRef.current?.input,
		},
	);

	return (
		<div {...props}>
			<Input
				ref={inputRef}
				allowClear
				value={value}
				prefix={<Icon name="i-lucide:search" />}
				size="small"
				placeholder={t("clipboard.hints.search_placeholder")}
				onCompositionStart={setTrue}
				onCompositionEnd={setFalse}
				onChange={(event) => {
					setValue(event.target.value);
				}}
			/>
		</div>
	);
};

export default Search;
