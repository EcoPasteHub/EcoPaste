import UnoIcon from "@/components/UnoIcon";
import type { InputRef } from "antd";
import { Input } from "antd";
import type { FC, HTMLAttributes } from "react";
import { MainContext } from "../..";

const SearchInput: FC<HTMLAttributes<HTMLDivElement>> = (props) => {
	const { rootState } = useContext(MainContext);
	const inputRef = useRef<InputRef>(null);
	const [value, setValue] = useState<string>();
	const [isComposition, { setTrue, setFalse }] = useBoolean();
	const { t } = useTranslation();

	useEffect(() => {
		if (isComposition) return;

		rootState.search = value;
	}, [value, isComposition]);

	useTauriFocus({
		onFocus() {
			const { search } = clipboardStore;

			// 搜索框默认聚焦
			if (search.defaultFocus) {
				inputRef.current?.focus();
			} else {
				inputRef.current?.blur();
			}
		},
		onBlur() {
			const { search } = clipboardStore;

			// 搜索框自动清空
			if (search.autoClear) {
				setValue(void 0);
			}
		},
	});

	useKeyPress(PRESET_SHORTCUT.SEARCH, () => {
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
				autoCorrect="off"
				allowClear
				value={value}
				prefix={<UnoIcon name="i-lucide:search" />}
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

export default SearchInput;
